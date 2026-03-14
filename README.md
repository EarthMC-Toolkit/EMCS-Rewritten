# EMCS-Rewritten
A rewritten version of the [EarthMC Stats](https://github.com/EarthMC-Toolkit/EarthMC-Stats) Discord bot in Go.\
This rewrite aims to make the bot self-hostable, with the [EMCS Rewritten](https://canary.discord.com/oauth2/authorize?client_id=656231016385478657) Discord bot being one of these instances. 

To keep the bot up 24/7, I recommend running the bot in `tmux` or even `pm2` which you can detach from on a VM.

## Why the rewrite?
I started this bot in hopes that it will be more powerful than **EMCS** with massive improvements to performance and stability.
The database will be local, so you are responsible for upkeep of data. For 24/7 uptime, use a VPS or self-host with a Raspberry Pi etc.

**EMCSRW** should be much easier to maintain and has been designed around the Official API mainly, rather than outdated map data.
While map data may still be used in certain cases or if the OAPI goes down, I believe this rewrite was necessary as the previous bot had too much
technical debt to make it worth the time and effort of updating, as well as the Firestore DB being severely limited with it being so popular as read/writes were almost always maxed out.

## Development
1. Clone this repository.
1. Create a Discord bot and put its **Client Token** in an `.env` file in the project root like so:

   	```sh 
    export BOT_TOKEN=yourTokenHere
    ```
1. Authorize and invite your bot to a guild or install it as a user app.
1. Finish configuring `.env` and run the bot (see next sections).

### Configuration
You will want to make sure you configure the bot to ensure everything works as intended.\
Configuring the bot is as simple as editing the `.env` file (may change in future).
```sh
export BOT_TOKEN=botTokenHere
export BOT_APP_ID=botAppIdHere
export DEV_ID=yourDiscordIdHere
export API_PORT=7777
```

### Running the bot
`go run . register` -> Uses a temporary Discord session to register commands, then exits the process immediately.
`go run . bot` -> Runs the bot and connects to Discord. The process runs until a panic or `Ctrl+C` (graceful exit).
`go run . api` -> Starts an API and listens to the port specified in `.env` (see next section).

To start immediately after registering, simply append it like so: `go run . register && go run . bot`

⚠️ You should only ever run `register` before `bot` - not after the bot already started!\
ℹ️ The bot uses a lock file, meaning only a single instance will exist across processes.\
ℹ️ The API should be run in a seperate session/process so it continues while the bot is down.

### Custom API
> [!WARNING]
> Only serve an API if you need one and know what you are doing.

By default, a custom API is not served. To serve one, simply add the following variables to your `.env` file.\
If you are not using a reverse proxy (see next section), you can access it at `localhost:<API_PORT>`.
```sh
export API_PORT=7777
```

#### Reverse Proxy
To serve to a domain instead of `localhost`, ensure you have a reverse proxy set up.
For example, here is a small `Caddyfile` you can use after you have installed **Caddy**.

> [!NOTE]
> If you encounter errors, ensure the following:
> - You have opened the port you specified in .env on your machine/VM.
> - You have ran `caddy reload --config /etc/caddy/Caddyfile` (or equivalent).
> - You have ran `sudo ufw 80`, `sudo ufw 443` and `sudo ufw enable` to allow HTTP/S traffic.
> - You have restarted your machine/VM after all of the above.
```sh
your.domain.com {
	# Set this path to your sites directory.
	# Keep commented unless you want to also serve a static website.
	# root * /usr/share/caddy

    # where the reverse proxy should listen.
	reverse_proxy localhost:7777

	# compression
	encode zstd gzip

	# headers perms. by default a GET request is allowed from any origin
	header {
        Access-Control-Allow-Methods GET
		Access-Control-Allow-Origin *
		# Access-Control-Allow-Headers *
	}
}
```
You can then access the API at `https://your.domain.com/<emcMapName>/<endpoint>`.
List of endpoints as of 16 Jan 2026:
- `alliances`
- `players`

## Project Structure
>- `main.go` -> Project entrypoint. Responsible for loading `env` and passing bot token to `bot.Run`.
>- `bot` -> Where the bot runs from. Contains all bot logic for commands, events etc.
>   - `events` -> The package where Discord event handlers like `OnReady` are run and are handled.
> 	- `scheduler` -> Task scheduler logic for running tasks at an interval which can gracefully shutdown.
> 	- `slashcommands` -> Self explanatory. Contains all slash commands as seperate files which handle their own execution.
>   - `bot.go` -> The file where the bot connects to Discord, also responsible for setting event handlers and intents.
>- `api` -> Contains packages relating to APIs. Contains funcs that interact with both where necessary.
>   - `mapi` -> For interacting with the map API. (Currently Squaremap)
>   - `oapi` -> For interacting with the Official API.
>   - `capi` -> Serves a Custom API using info from the `database` package. NOT REQUIRED IF FORKING.
>- `database` -> For all code that relates to or interacts with a DB or store/cache.
>	- `store` -> For interacting with stores themselves after retreiving them from the database.
>- `db` -> Where permanent data such as alliances are intended to be stored. Git ignored.
>- `shared` -> For things that can be shared, e.g. constants or embed related funcs/vars.
>- `utils` -> Contains packages for reusable funcs like helpers for strings, slices, http, logging etc.

## Contributing
If you know **Golang** and the basics of the **discordgo** library, I encourage you to create pull requests or suggest features.
You can also fork this project and use it as a base if you so desire, but the GPL license requires you to keep the source code available.

Creating a new command is as simple as adding it to `RegisterAllCommands()` and then creating a file which satisfies the `SlashCommand` interface (see code below). Finally, run the command `go run . register && go run . bot` and refresh Discord (Ctrl+R).

```go
type ExampleCommand struct{}

func (cmd ExampleCommand) Name() string { return "example" }
func (cmd ExampleCommand) Description() string {
	return "This is an example description for a slash command."
}

func (cmd ExampleCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about XYZ.",
			Options: AppCommandOpts{
				discordutil.AutocompleteStringOption("name", "The name of the thing to query.", 2, 36, true),
			},
		},
	}
}

// Allows this command to handle its own execution once registered (via an interface).
func (cmd ExampleCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	// Indicates that the command will take >3s so it does not timeout.
	// Displays the "bot is thinking..." text.
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

 	// Do some stuff according to chosen subcmd.
	// If no subcommands exist, we can skip the option and execute.
	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("query"); opt != nil {
		nationNameArg := opt.GetOption("name").StringValue()
		_, err := executeExampleQueryFunc(s, i.Interaction, nationNameArg)
		return err
	}

	return nil
}

// =============================================================================================
// The following are optional and can be removed if desired.
// You can find example usage of these across files within the `./bot/slashcommands/` directory.

func (cmd ExampleCommand) HandleModal(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
	return nil
}

func (cmd ExampleCommand) HandleButton(s *discordgo.Session, i *discordgo.Interaction, customID string) error {
	return nil
}

func (cmd ExampleCommand) HandleAutocomplete(s *discordgo.Session, i *discordgo.Interaction) error {
	return nil
}

// =============================================================================================
```