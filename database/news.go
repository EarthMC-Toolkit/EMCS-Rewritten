package database

import (
	"regexp"
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

func MessagesToNewsEntries(msgs []*discordgo.Message) map[string]NewsEntry {
	// msgs should already exclude deleted ones, so we don't need to check for those here.
	entries := make(map[string]NewsEntry, len(msgs))
	for _, msg := range msgs {
		content := strings.TrimSpace(msg.Content)
		if strings.HasPrefix(content, "<@") && strings.HasSuffix(content, ">") {
			continue // mentions without headlines aren't news. thx EMCL.
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
