module emcsrw

go 1.25.3

// discordgo doesn't do regular releases. point to my fork to use new features
replace github.com/bwmarrin/discordgo => github.com/owen3h/discordgo v0.0.0-20260214123928-f43dd94faaac

require (
	github.com/bwmarrin/discordgo v0.29.0
	github.com/fatih/color v1.19.0
	github.com/joho/godotenv v1.5.1
	github.com/samber/lo v1.53.0
	github.com/sanity-io/litter v1.5.8
	github.com/yuin/goldmark v1.8.5
)

require (
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/rogpeppe/go-internal v1.15.0
	github.com/rs/cors v1.11.1
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0
	golang.org/x/time v0.15.0
)
