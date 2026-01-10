package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"emcsrw/shared"
	"emcsrw/shared/embeds"
	"emcsrw/utils"
	"emcsrw/utils/discordutil"
	"emcsrw/utils/geometry"
	"fmt"
	"math"

	"github.com/bwmarrin/discordgo"
)

type MapBounds struct {
	Left, Right, Top, Bottom float64
}

// TODO: Move to `shared` package?
var MAP_BOUNDS = MapBounds{
	Left:   -33280,
	Right:  33080,
	Top:    -16640,
	Bottom: 16508,
}

type RouteCommand struct{}

func (cmd RouteCommand) Name() string { return "route" }
func (cmd RouteCommand) Description() string {
	return "Get the optimal distance, direction and closest spawn for a given point."
}

func (cmd RouteCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "fastest",
			Description: "Retrieve the most optimal route, without filtering out flags like PVP.",
			Options: AppCommandOpts{
				discordutil.RequiredNumberOption("x", "The map coordinate on the X axis (left/right).", MAP_BOUNDS.Left, MAP_BOUNDS.Right),
				discordutil.RequiredNumberOption("z", "The map coordinate on the Z axis (top/bottom).", MAP_BOUNDS.Top, MAP_BOUNDS.Bottom),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "safest",
			Description: "Retrieve the safest route, avoiding PVP enabled towns.",
			Options: AppCommandOpts{
				discordutil.RequiredNumberOption("x", "The map coordinate on the X axis (left/right).", MAP_BOUNDS.Left, MAP_BOUNDS.Right),
				discordutil.RequiredNumberOption("z", "The map coordinate on the Z axis (top/bottom).", MAP_BOUNDS.Top, MAP_BOUNDS.Bottom),
			},
		},
	}
}

func (cmd RouteCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return err
	}

	townStore, err := database.GetStore(mdb, database.TOWNS_STORE)
	if err != nil {
		return err
	}

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	cdata := i.ApplicationCommandData()
	opt, safe := cdata.GetOption("fastest"), false
	if opt == nil {
		opt, safe = cdata.GetOption("safest"), true
	}

	x := opt.GetOption("x").FloatValue()
	z := opt.GetOption("z").FloatValue()

	inputLoc := oapi.Location2D{
		X: float32(x),
		Z: float32(z),
	}

	r, err := getRoute(inputLoc, safe, townStore, nationStore)
	if err != nil {
		return err
	}

	ct, cn := r.ClosestTown, r.ClosestNation

	title := fmt.Sprintf("Route to %d, %d | Fastest", int(x), int(z))
	desc := "Showing the optimal spawns based on these factors:\n**Can Outsiders Spawn**: On\n**Public**: On\n**PVP**: Any\n"

	if safe {
		title = fmt.Sprintf("Route to %d, %d | Safest", int(x), int(z))
		desc = "Showing the optimal spawns based on these factors:\n**Can Outsiders Spawn**: On\n**Public**: On\n**PVP**: Off\n"
	}

	ctSpawn := ct.Entity.Spawn()
	cnSpawn := cn.Entity.Spawn()

	ctName := fmt.Sprintf("[%s](https://map.earthmc.net?x=%f&z=%f&zoom=5)", ct.Entity.Name, ctSpawn.X, ctSpawn.Z)
	cnName := fmt.Sprintf("[%s](https://map.earthmc.net?x=%f&z=%f&zoom=4)", cn.Entity.Name, cnSpawn.X, cnSpawn.Z)

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       utils.HexToInt("#DE3163"),
		Footer:      embeds.DEFAULT_FOOTER,
		Fields: []*discordgo.MessageEmbedField{
			NewEmbedField("Closest Town", formatRouteTarget(ctName, ct), true),
			NewEmbedField("Closest Nation", formatRouteTarget(cnName, cn), true),
		},
	}

	_, err = discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
		Embeds: []*discordgo.MessageEmbed{embed},
	})

	return err
}

type Route struct {
	ClosestTown   RouteTarget[oapi.TownInfo]
	ClosestNation RouteTarget[oapi.NationInfo]
}

type RouteTarget[T any] struct {
	Entity      T
	Distance    float64
	Direction   string
	TravelTimes *TravelTimes
}

// Gets the optimal route, regardless of PVP flag.
func getRoute(
	loc oapi.Location2D, safe bool,
	townStore *store.Store[oapi.TownInfo],
	nationStore *store.Store[oapi.NationInfo],
) (*Route, error) {
	var closestTown RouteTarget[oapi.TownInfo]
	var closestNation RouteTarget[oapi.NationInfo]

	var minTownDist = math.MaxFloat64
	var minNationDist = math.MaxFloat64

	townStore.ForEach(func(_ store.StoreKey, t oapi.TownInfo) {
		// if !t.Status.Public {
		// 	return true // This is just for other nation members to spawn i think
		// }
		if !t.Status.CanOutsidersSpawn {
			return // skip if we cannot spawn at this town
		}
		if safe && t.Perms.Flags.PVP {
			return // skip if PVP enabled and using "safest" subcmd
		}

		spawn := t.Spawn().Location2D
		dist := distanceBetween(spawn, loc)
		if dist < minTownDist {
			minTownDist = dist
			closestTown = RouteTarget[oapi.TownInfo]{
				Entity:      t,
				Distance:    dist,
				Direction:   cardinalDirection(loc.X, loc.Z, spawn.X, spawn.Z, true),
				TravelTimes: calcTravelTimes(dist),
			}
		}
	})

	nationStore.ForEach(func(_ store.StoreKey, n oapi.NationInfo) {
		if !n.Status.Public {
			return // cant spawn, skip
		}

		spawn := n.Spawn().Location2D
		dist := distanceBetween(spawn, loc)
		if dist < minNationDist {
			minNationDist = dist
			closestNation = RouteTarget[oapi.NationInfo]{
				Entity:      n,
				Distance:    dist,
				Direction:   cardinalDirection(loc.X, loc.Z, spawn.X, spawn.Z, true),
				TravelTimes: calcTravelTimes(dist),
			}
		}
	})

	return &Route{
		ClosestTown:   closestTown,
		ClosestNation: closestNation,
	}, nil
}

var DIRECTIONS = []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
var BASE_DIRECTIONS = []string{"N", "E", "S", "W"}

func cardinalDirection(originX, originZ, destX, destZ float32, allowIntermediates bool) string {
	dx := float64(destX - originX)
	dz := float64(destZ - originZ)

	angle := math.Atan2(dz, dx) * 180 / math.Pi // arc tangent of z/x in radians, converted to degrees
	normalized := math.Mod(angle+90+360, 360)   // normalize angle to [0, 360)
	if allowIntermediates {
		index := int(math.Round(normalized/45)) % 8
		return DIRECTIONS[index]
	} else {
		index := int(math.Round(normalized/90)) % 4
		return BASE_DIRECTIONS[index]
	}
}

type TravelTimes struct {
	Sneaking  int
	Walking   int
	Sprinting int
	Boat      int
}

// example constants for movement speeds (blocks per minute)
type ActionSpeeds struct {
	Sneak, Walk, Sprint, Boat float64
}

var ACTION_SPEEDS = ActionSpeeds{
	Sneak:  1.295,
	Walk:   4.317,
	Sprint: 5.612,
	Boat:   8.0,
}

func calcTravelTimes(distance float64) *TravelTimes {
	if distance <= 0 {
		return nil
	}

	return &TravelTimes{
		Sneaking:  int(distance / ACTION_SPEEDS.Sneak / 60.0), // sec → min
		Walking:   int(distance / ACTION_SPEEDS.Walk / 60.0),
		Sprinting: int(distance / ACTION_SPEEDS.Sprint / 60.0),
		Boat:      int(distance / ACTION_SPEEDS.Boat / 60.0),
	}
}

// Returns the manhattan distance between two sets of 2D points (X, Y).
func distanceBetween(loc1 oapi.Location2D, loc2 oapi.Location2D) float64 {
	return geometry.ManhattanDistance2D(
		float64(loc1.X), float64(loc2.X),
		float64(loc1.Z), float64(loc2.Z),
	)
}

func formatRouteTarget[T any](name string, rt RouteTarget[T]) string {
	if rt.TravelTimes == nil {
		return fmt.Sprintf("%s\nDistance: `%d` blocks\nDirection: `%s`", name, int(rt.Distance), rt.Direction)
	}

	return fmt.Sprintf(
		"%s\n\nDistance: `%d` blocks\nDirection: `%s`\nTravel Times:\n• Sneak: `%d` min\n• Walk: `%d` min\n• Sprint: `%d` min\n• Boat: `%d` min",
		name, int(rt.Distance), rt.Direction,
		rt.TravelTimes.Sneaking,
		rt.TravelTimes.Walking,
		rt.TravelTimes.Sprinting,
		rt.TravelTimes.Boat,
	)
}
