package database

import (
	"cmp"
	"emcsrw/api/oapi"
	"emcsrw/database/store"
	"emcsrw/utils/sets"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var IMAGE_REGEX = regexp.MustCompile(`(https?:\/\/.*\.(?:png|jpg|jpeg|gif|webp)(?:\?[^\s]*)?)`)
var BOLD_REGEX = regexp.MustCompile(`\*\*(.*?)\*\*`)

var NTIMES_LOGO = ":nt:1488052397946310696"
var NTIMES_LOGO_REPLACER = strings.NewReplacer(
	"<"+NTIMES_LOGO+">", "",
	"\u003C"+NTIMES_LOGO+"\u003E", "",
	"\u003c"+NTIMES_LOGO+"\u003e", "",
)

// var EMCL_LOGO = ":EMCL:12345678910"
// var EMCL_LOGO_REPLACER = strings.NewReplacer(
// 	"<"+EMCL_LOGO+">", "",
// 	"\u003C"+EMCL_LOGO+"\u003E", "",
// )

type NewsEntry struct {
	//ID        string   `json:"id"`
	Message   string   `json:"message"`
	Headline  string   `json:"headline"`
	Images    []string `json:"images"`
	Timestamp int64    `json:"timestamp"`
}

// Returns a new string with the headline in bold text and the news provider's emoji before it.
func (e *NewsEntry) ParsedHeadline() string {
	return fmt.Sprintf("<%s> **%s**", NTIMES_LOGO, e.Headline)
}

// Returns a new string with attached image hyperlinks in brackets and a relative Discord timestamp attached at the end.
// If no images images are present in this news entry, only the timestamp is returned.
//
// Examples:
//
//	"7 days ago"
//	"(Image) 7 days ago"
//	"(Image, Image) 7 days ago"
func (e *NewsEntry) Context() string {
	timestamp := fmt.Sprintf("<t:%d:R>", e.Timestamp/1000)
	if len(e.Images) == 0 {
		return timestamp
	}

	imgs := make([]string, len(e.Images))
	for j, img := range e.Images {
		imgs[j] = fmt.Sprintf("[Image](%s)", img)
	}

	imgLinks := strings.Join(imgs, ", ")
	return fmt.Sprintf(" (%s) %s", imgLinks, timestamp)
}

func NewNewsEntry(msg *discordgo.Message) NewsEntry {
	entry := NewsEntry{
		//ID:        msg.ID,
		Message:   msg.Content,
		Timestamp: msg.Timestamp.UnixMilli(),
	}

	// If there are attachments in the message, push attachment.url to images
	for _, attachment := range msg.Attachments {
		if IMAGE_REGEX.MatchString(attachment.URL) {
			entry.Images = append(entry.Images, attachment.URL)
		}
	}

	// Strip the news logo so it isn't included when we go to set the headline.
	cleanedMsg := NTIMES_LOGO_REPLACER.Replace(entry.Message)

	// Content has at least one image link.
	if matches := IMAGE_REGEX.FindAllString(msg.Content, -1); len(matches) > 0 {
		entry.Images = append(entry.Images, matches...)
		cleanedMsg = strings.TrimSpace(IMAGE_REGEX.ReplaceAllString(cleanedMsg, "")) // TODO: is this is necessary if we match bold anyway
	}

	entry.Headline = extractHeadline(cleanedMsg)
	return entry
}

// Extracts the headline from the message, where the headline is one of two:
//
//  1. Everything in bold (between first set of double asterisks).
//  2. Everything after the news emoji if markdown '#' heading(s) are present.
//
// In both cases we only use first line, everything on lines after that gets removed (extra context, credit etc).
func extractHeadline(msg string) string {
	if matches := BOLD_REGEX.FindStringSubmatch(msg); len(matches) > 1 {
		return strings.TrimSpace(matches[1]) // We found text in bold, just return everything inside :)
	}

	before, _, _ := strings.Cut(msg, "\n")                   // Ensure we strip extra fluff that might be on a new line.
	return strings.TrimSpace(strings.TrimLeft(before, "# ")) // Remove all markdown headings, we just need the headline text.
}

type NewsMessageID = string

// Converts Discord messages into a map of [NewsEntry] (headline, timestamp etc) keyed by the message ID.
// It filters out invalid messages (non-news, mentions-only, missing logo) and the output contains only unique headlines.
func MessagesToNewsEntries(msgs []*discordgo.Message) map[NewsMessageID]NewsEntry {
	entries := make(map[NewsMessageID]NewsEntry, len(msgs))
	seen := sets.New[string]() // headline tracker to remove dupes

	// msgs should already exclude deleted ones, so we don't need to check for those here.
	for _, msg := range msgs {
		content := strings.TrimSpace(msg.Content)
		if strings.HasPrefix(content, "<@") && strings.HasSuffix(content, ">") {
			continue // mentions without headlines aren't news.
		}
		if !strings.Contains(content, NTIMES_LOGO) {
			continue // message without a logo probably isn't news.
		}

		entry := NewNewsEntry(msg)
		key := strings.ToLower(entry.Headline)
		if seen.Has(key) {
			continue
		}

		entries[msg.ID] = entry
		seen.Append(key)
	}

	return entries
}

// Finds all news entries within newsStore that explicitly mention nation by its name.
//
// Matching is case-insensitive and uses whole-word boundaries, preventing cases such as "Mali" matching "Somali".
// Underscores in nation names are treated as spaces so both "New_York" and "New York" match.
//
// Results are sorted by newest-first.
func GetNationNews(newsStore *store.Store[NewsEntry], nation oapi.NationInfo) []NewsEntry {
	name := strings.ToLower(strings.ReplaceAll(nation.Name, "_", " "))
	re := makePattern(name)

	nationNews := newsStore.FindAll(func(e NewsEntry) bool {
		return re.MatchString(strings.ToLower(e.Headline))
	})
	slices.SortFunc(nationNews, func(a, b NewsEntry) int {
		return cmp.Compare(b.Timestamp, a.Timestamp)
	})

	return nationNews
}

// Finds all news entries that mention the alliance by either its full name (Label) or its short identifier (e.g. "PLC").
//
// Matching is case-insensitive and uses whole-word boundaries to avoid false positives,
// such as matching identifiers embedded inside larger words.
//
// Results are sorted by newest-first.
func GetAllianceNews(newsStore *store.Store[NewsEntry], alliance Alliance) []NewsEntry {
	labelRgx := makePattern(alliance.Label)
	identRgx := makePattern(alliance.Identifier)

	allianceNews := newsStore.FindAll(func(e NewsEntry) bool {
		headline := strings.ToLower(e.Headline)
		matchesLabel := (labelRgx != nil && labelRgx.MatchString(headline))
		matchesIdent := (identRgx != nil && identRgx.MatchString(headline))
		return matchesLabel || matchesIdent
	})
	slices.SortFunc(allianceNews, func(a, b NewsEntry) int {
		return cmp.Compare(b.Timestamp, a.Timestamp)
	})

	return allianceNews
}

// Returns a regex that matches s as a whole word, preventing substring matches.
//
// The function does the following:
//   - Escapes regex special characters so the input is treated literally
//   - Wraps the result in \b word boundaries to enforce whole-word matching.
//
// Example patterns:
//
//	"PLC"   -> \bPLC\b
//	"Mali"  -> \bMali\b
func makePattern(s string) *regexp.Regexp {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return nil
	}

	return regexp.MustCompile(`\b` + regexp.QuoteMeta(s) + `\b`)
}
