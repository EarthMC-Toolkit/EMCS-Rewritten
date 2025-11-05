package discordutil

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const DEV_ID = "263377802647175170"

func IsDev(i *discordgo.Interaction) bool {
	author := GetInteractionAuthor(i)
	return author.ID == DEV_ID
}

func HasRole(m *discordgo.Member, roleID string) (bool, error) {
	if m == nil {
		return false, fmt.Errorf("cannot get roles from interaction: Member is nil")
	}

	return slices.Contains(m.Roles, roleID), nil
}

// Returns a Discord invite code by parsing the input string.
//
// If input is a link, this func validates that it is a correct Discord domain, and then extracts the code from it.
// Otherwise, the input is assumed to be a code already and as long as it is between 1-32 chars, it will remain unchanged.
func ExtractInviteCode(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty input")
	}

	// Try parsing as URL
	u, err := url.Parse(input)
	if err == nil && u.Host != "" {
		host := strings.ToLower(u.Host)
		path := strings.Trim(u.Path, "/")

		switch host {
		case "discord.gg", "www.discord.gg":
			if path == "" {
				return "", fmt.Errorf("no code found in invite URL")
			}

			return path, nil
		case "discord.com", "www.discord.com":
			if postfix, ok := strings.CutPrefix(path, "invite/"); ok {
				code := postfix
				if code == "" {
					return "", fmt.Errorf("no code found in invite URL")
				}

				return code, nil
			}
		}
	}

	code := strings.TrimSpace(input)
	if code == "" || len(code) > 32 {
		return "", fmt.Errorf("invalid Discord invite code")
	}

	return code, nil
}

func ValidateInviteCode(code string, s *discordgo.Session) (*discordgo.Invite, error) {
	invite, err := s.Invite(code)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invite: %w", err)
	}
	if invite == nil || invite.Code == "" {
		return nil, fmt.Errorf("nil invite or empty code")
	}

	// Permanent invites are indicated by ExpiresAt == nil.
	if invite.ExpiresAt != nil {
		return invite, fmt.Errorf("invite is not permanent")
	}

	return invite, nil
}
