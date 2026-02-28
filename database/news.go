package database

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var IMAGE_REGEX = regexp.MustCompile(`(https?:\/\/.*\.(?:png|jpg|jpeg|gif|webp)(?:\?[^\s]*)?)`)
var BOLD_REGEX = regexp.MustCompile(`\*\*(.*?)\*\*`)

var TBI_LOGO_REPLACER = strings.NewReplacer(
	"<:TBI:1071887302676185241>", "",
	"\u003C:TBI:1071887302676185241\u003E", "",
)

type NewsEntry struct {
	ID        string   `json:"id"`
	Message   string   `json:"message"`
	Headline  string   `json:"headline"`
	Images    []string `json:"images"`
	Timestamp int64    `json:"timestamp"`
}

func NewNewsEntry(msg *discordgo.Message) NewsEntry {
	entry := NewsEntry{
		ID:        msg.ID,
		Message:   msg.Content,
		Timestamp: msg.Timestamp.UnixMilli(),
	}

	// If there are attachments in the message, push attachment.url to images
	for _, attachment := range msg.Attachments {
		if IMAGE_REGEX.MatchString(attachment.URL) {
			entry.Images = append(entry.Images, attachment.URL)
		}
	}

	// Ensure TBI logo is gone.
	headline := TBI_LOGO_REPLACER.Replace(entry.Message)

	// Content has at least one image link.
	if matches := IMAGE_REGEX.FindAllString(msg.Content, -1); len(matches) > 0 {
		entry.Images = append(entry.Images, matches...)
		headline = strings.TrimSpace(IMAGE_REGEX.ReplaceAllString(headline, ""))
	}

	entry.Headline = extractHeadline(headline)
	return entry
}

func MessagesToNewsEntries(msgs []*discordgo.Message) []NewsEntry {
	entries := make([]NewsEntry, 0, len(msgs))
	for _, msg := range msgs {
		entries = append(entries, NewNewsEntry(msg))
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
