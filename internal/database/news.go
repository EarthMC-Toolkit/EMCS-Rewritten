package database

import (
	"cmp"
	"emcsrw/internal/database/store"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils/sets"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var IMAGE_REGEX = regexp.MustCompile(`(https?:\/\/.*\.(?:png|jpg|jpeg|gif|webp)(?:\?[^\s]*)?)`)
var BOLD_REGEX = regexp.MustCompile(`\*\*(.*?)\*\*`)
var TAGS_REGEX = regexp.MustCompile("Tags:\\s*`([^`]*)`")

var NTIMES_EMOJI = ":nt:1488052397946310696"
var NTIMES_BOT_EMOJI = ":emoji:1488052397946310696"
var NTIMES_EMOJI_REPLACER = strings.NewReplacer(
	"<"+NTIMES_EMOJI+">", "",
	"\u003C"+NTIMES_EMOJI+"\u003E", "",
	"\u003c"+NTIMES_EMOJI+"\u003e", "",
	"<"+NTIMES_BOT_EMOJI+">", "",
	"\u003C"+NTIMES_BOT_EMOJI+"\u003E", "",
	"\u003c"+NTIMES_BOT_EMOJI+"\u003e", "",
)

// var EMCL_LOGO = ":EMCL:12345678910"
// var EMCL_LOGO_REPLACER = strings.NewReplacer(
// 	"<"+EMCL_LOGO+">", "",
// 	"\u003C"+EMCL_LOGO+"\u003E", "",
// )

// Generic entry representing a news message. This can be used in both cases where news reports were either
// sent by the NT bot or manually by a reporter. When sent manually, tags and reporter name may be missing.
//
// The only difference between the two is that NT bot messages will always
// have the NT logo in the message, while reporter messages *may* not.
type NewsEntry struct {
	// The raw Discord message contents, which may include the news logo, bold text, image links, tags, credit, reporter.
	Message string `json:"message"`
	// Extracted from the message content, either from bold text or from the first line after the news logo.
	Headline string `json:"headline"`
	// Extracted from both message attachments and any image links in the message content.
	Images sets.Set[string] `json:"images"`
	// The timestamp denoting when the message was posted (ms since the last Unix epoch).
	Timestamp int64 `json:"timestamp"`
	// Single comma-seperated string of tags, if any exist in the msg. These are usually only present for reports sent by the bot.
	Tags string `json:"tags,omitempty"`
}

// Returns a new string with the headline in bold text and the news provider's emoji before it.
func (e *NewsEntry) ParsedHeadline() string {
	return fmt.Sprintf("<%s> **%s**", NTIMES_EMOJI, e.Headline)
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

	imgs := e.Images.KeysFunc(func(img string) string {
		return fmt.Sprintf("[Image](%s)", img)
	})

	imgLinks := strings.Join(imgs, ", ")
	return fmt.Sprintf(" (%s) %s", imgLinks, timestamp)
}

func NewNewsEntry(m *discordgo.Message) NewsEntry {
	e := NewsEntry{
		Message:   m.Content,
		Timestamp: m.Timestamp.UnixMilli(),
	}

	return ParseEntry(e, m)
}

func ParseEntry(entry NewsEntry, m *discordgo.Message) NewsEntry {
	// All attachments that are images should be added to the entry.Images slice,
	// but any duplicate images found in the message content should be ignored since they are already included.
	for _, attachment := range m.Attachments {
		urlStr := attachment.URL
		if IMAGE_REGEX.MatchString(urlStr) {
			u, err := url.Parse(urlStr)
			if err == nil {
				u.Fragment = ""
				urlStr = u.String()
			}
			entry.Images.Add(urlStr)
		}
	}

	// Strip the news logo so it isn't included when we go to set the headline.
	cleanedMsg := NTIMES_EMOJI_REPLACER.Replace(entry.Message)

	// Content has at least one image link.
	if matches := IMAGE_REGEX.FindAllString(m.Content, -1); len(matches) > 0 {
		entry.Images.Add(matches...)
		cleanedMsg = strings.TrimSpace(IMAGE_REGEX.ReplaceAllString(cleanedMsg, "")) // TODO: is this is necessary if we match bold anyway?
	}

	entry.Headline = extractHeadline(cleanedMsg)
	entry.Tags = extractTags(cleanedMsg)

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

func extractTags(msg string) string {
	if matches := TAGS_REGEX.FindStringSubmatch(msg); len(matches) > 1 {
		return strings.TrimSpace(matches[1]) // Match [0] is the full string, [1] is the captured group ("alliance, conflict" etc).
	}

	return ""
}

type NewsMessageID = string

// Converts Discord messages into a map of [NewsEntry] (headline, timestamp etc) keyed by the message ID.
// It filters out invalid messages (non-news, mentions-only, missing logo) and the output contains only unique headlines.
func MessagesToNewsEntries(s *discordgo.Session, msgs []*discordgo.Message) map[NewsMessageID]NewsEntry {
	entries := make(map[NewsMessageID]NewsEntry, len(msgs))
	seen := sets.New[string]() // headline tracker to remove dupes

	// msgs should already exclude deleted ones, so we don't need to check for those here.
	for _, msg := range msgs {
		content := strings.TrimSpace(msg.Content)
		if strings.HasPrefix(content, "<@") && strings.HasSuffix(content, ">") {
			continue // mentions without headlines aren't news.
		}
		if !strings.Contains(content, NTIMES_EMOJI) && !strings.Contains(content, NTIMES_BOT_EMOJI) {
			continue // message without a logo emoji probably isn't news.
		}

		entry := NewNewsEntry(msg)
		key := strings.ToLower(entry.Headline)
		if seen.Has(key) {
			continue
		}

		entries[msg.ID] = entry
		seen.Add(key)
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
