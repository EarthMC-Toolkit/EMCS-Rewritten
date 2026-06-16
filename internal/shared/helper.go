package shared

import (
	"emcsrw/internal/database"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils/discordutil"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

// NOTE: Potential import cycle. Consider just duplicating necessary funcs rather than importing discordutil.
var NewEmbedField = discordutil.NewEmbedField
var PrependField = discordutil.PrependField
var AddField = discordutil.AddField

// Returns a circular emoji equivalent to the value of v
// where true becomes a green check, false a red cross.
func BoolToEmoji(v bool) string {
	if v {
		return EMOJIS.CIRCLE_CHECK
	}

	return EMOJIS.CIRCLE_CROSS
}

// Returns a single string representing a list of player names with their affiliations.
// For example:
//
//	`Player1` of Town1 (**Nation1**)
//	`Player2` of Town2
//	`Player3` (Townless)
func BuildAffiliationsString(players []database.BasicPlayer) string {
	if players == nil {
		return ""
	}

	lines := []string{}
	for _, p := range players {
		line := fmt.Sprintf("`%s`", p.Name)
		if p.Town != nil {
			line += fmt.Sprintf(" of %s", p.Town.Name)
			if p.Nation != nil {
				line += fmt.Sprintf(" (**%s**)", p.Nation.Name)
			}
		} else {
			line += " (Townless)"
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// Given a slice of news entries, this func builds a string to include up to `max` news headlines split by \n\n.
//
// Example of a single headline:
//
//	"<logo> **Headline** 4 days ago"
//	"<logo> **Headline** <newline> (Image) 4 days ago"
//	"<logo> **Headline** <newline> (Image, Image) 4 days ago"
func BuildNewsString(news []database.NewsEntry, max uint8, charLimit uint16) (string, uint8) {
	b := strings.Builder{}
	count := uint8(0)
	size := 0

	for i, entry := range news {
		if i == int(max) {
			break
		}

		newsStr := entry.ParsedHeadline() // The headline but in bold text and including the logo.
		ctx := entry.Context()            // Attached images and discord formatted timestamp.

		sep := " "
		if len(entry.Images) > 0 {
			sep = "\n"
		}

		part := newsStr + sep + ctx
		if count > 0 {
			part = "\n\n" + part
		}

		if size+len(part) > int(charLimit) {
			break
		}

		b.WriteString(part)
		size += len(part)
		count++
	}

	return b.String(), count
}

func BuildNationRanksString(nation oapi.NationInfo) string {
	hasPlayers := false

	lines := make([]string, 0, len(nation.Ranks))
	for rank, players := range nation.Ranks {
		if len(players) == 0 {
			continue
		}

		names := lo.Map(players, func(e oapi.Entity, _ int) string {
			return fmt.Sprintf("`%s`", e.Name)
		})

		line := fmt.Sprintf("[%d] %s\n%s", len(players), rank, strings.Join(names, ", "))
		lines = append(lines, line)

		hasPlayers = true
	}

	if hasPlayers {
		return strings.Join(lines, "\n\n")
	}

	lines = lines[:0]
	for rank := range nation.Ranks {
		lines = append(lines, fmt.Sprintf("[0] %s", rank))
	}

	// Output all as [0] Chancellor, [0] Diplomat etc.
	return strings.Join(lines, "\n")
}

type SkinType string

const (
	SkinTypeFace2D      SkinType = "face"      // A simple 2D face. The helmet is rendered slightly larger than the face.
	SkinTypeFront2D     SkinType = "front"     // A square waist-up flat render of the player.
	SkinTypeFrontFull2D SkinType = "frontfull" // A flat render of the entire player.
	SkinTypeHead3D      SkinType = "head"      // Just the player's severed head.
	SkinTypeBust3D      SkinType = "bust"      // A square waist-up render of the player.
	SkinTypeFull3D      SkinType = "full"      // A render of the entire player.
)

type SkinFormat string

const (
	SkinFormatPNG  SkinFormat = "png"
	SkinFormatJXL  SkinFormat = "jxl"
	SkinFormatWEBP SkinFormat = "webp"
)

type SkinBuilder struct {
	URL   *url.URL
	Query url.Values
}

// subject can be a name or uuid
func NewSkinBuilder(subject string, skinType SkinType, format SkinFormat, size uint16) (*SkinBuilder, error) {
	skinURL, err := url.Parse(fmt.Sprintf("%s/%s/%d/%s.%s", SKIN_URL, string(skinType), size, subject, string(format)))
	if err != nil {
		return nil, err
	}

	return &SkinBuilder{
		URL:   skinURL,
		Query: skinURL.Query(),
	}, nil
}

func (sb *SkinBuilder) SetLossless() *SkinBuilder {
	sb.Query.Add("quality", "lossless")
	return sb
}

func (sb *SkinBuilder) SetAngle(yaw, pitch, roll int16) *SkinBuilder {
	if yaw != 0 {
		sb.Query.Set("y", strconv.Itoa(int(yaw)))
	}
	if pitch != 0 {
		sb.Query.Set("p", strconv.Itoa(int(pitch)))
	}
	if roll != 0 {
		sb.Query.Set("r", strconv.Itoa(int(roll)))
	}
	return sb
}

func (sb *SkinBuilder) Build() string {
	sb.URL.RawQuery = sb.Query.Encode()
	return sb.URL.String()
}
