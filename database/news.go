package database

import (
	"cmp"
	"emcsrw/api/oapi"
	"emcsrw/database/store"
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

	// Ensure the news logo is not included in the headline.
	headline := NTIMES_LOGO_REPLACER.Replace(entry.Message)

	// Content has at least one image link.
	if matches := IMAGE_REGEX.FindAllString(msg.Content, -1); len(matches) > 0 {
		entry.Images = append(entry.Images, matches...)
		headline = strings.TrimSpace(IMAGE_REGEX.ReplaceAllString(headline, ""))
	}

	entry.Headline = strings.TrimSpace(extractHeadline(headline))
	return entry
}

type NewsMessageID = string

func MessagesToNewsEntries(msgs []*discordgo.Message) map[NewsMessageID]NewsEntry {
	// msgs should already exclude deleted ones, so we don't need to check for those here.
	entries := make(map[NewsMessageID]NewsEntry, len(msgs))
	for _, msg := range msgs {
		content := strings.TrimSpace(msg.Content)
		if strings.HasPrefix(content, "<@") && strings.HasSuffix(content, ">") {
			continue // mentions without headlines aren't news.
		}
		if !strings.Contains(content, NTIMES_LOGO) {
			continue // message without a logo probably isn't news.
		}

		entries[msg.ID] = NewNewsEntry(msg)
	}

	return entries
}

// The headline is everything in bold (between first set of **), or before first new line.
func extractHeadline(s string) string {
	if matches := BOLD_REGEX.FindStringSubmatch(s); len(matches) > 1 {
		return matches[1]
	}

	before, _, _ := strings.Cut(s, "\n")
	return before
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
