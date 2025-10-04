# EMCS-Rewritten
A rewritten version of the [EarthMC Stats](https://github.com/EarthMC-Toolkit/EarthMC-Stats) Discord bot in Go.

## Why the rewrite?
I started this bot in hopes that it will be more powerful than **EMCS** with massive improvements to performance and stability.\
The database will be local, so you are responsible for upkeep of data. For 24/7 uptime, use a VPS or self-host with a Raspberry Pi etc.

With this being a fresh project, it allows me to be develop without introducing downtime or potential bugs to the current **EMCS** bot.

**EMCSRW** should be much easier to maintain and has been designed around the Official API mainly, rather than outdated map data.\
While map data may still be used in certain cases or if the OAPI goes down, I believe this rewrite was necessary as the previous bot had too much
technical debt to make it worth the time and effort of updating, as well as the Firestore DB being severely limited with it being so popular as read/writes were almost always maxed out.

## Development
1. Clone this repository.
1. Create a Discord bot and put its **Client Token** in an `.env` file in the project root like so:
    ```console 
    export BOT_TOKEN=yourTokenHere
    ```

1. Authorize and invite your bot to a guild or install it as a user app.
1. Start the bot with `go run main.go`.

## Contributing
If you know **Golang** and the basics of the **discordgo** library, I encourage you to create pull requests or suggest features. Now is probably the best time to contribute as the project is fresh and there is virtually no technical debt yet. There are already a few slash commands I've made that you can use as an example.

You can also fork this project and use it as a base if you so desire, but the GPL license requires you to keep the source code available.

## Project Structure
>- `main.go` -> Project entrypoint. Responsible for loading `env` and passing bot token to `bot.Run`.
>- `bot` -> Where the bot runs from. Contains all bot logic for commands, events, discord and db init etc.
>   - `bot.go` -> The file the bot actually runs from, responsible for setting up discord, opening connections and handling bot exit.
>   - `common` -> For things that can be shared like constants or embed build funcs.
>   - `store` -> For all code that relates to or interacts with a DB or store/cache.
>- `api` -> Contains packages relating to APIs. Contains funcs that interact with both where necessary.
>   - `mapi` -> For interacting with the map API. (Currently Squaremap)
>   - `oapi` -> For interacting with the Official API.
>- `db` -> Where permanent data such as alliances are intended to be stored. Git ignored.
>- `utils` -> Provides small funcs generally re-used a lot like helpers for strings, slices, http, logging etc.