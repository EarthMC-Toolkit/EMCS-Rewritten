# EMCS-Rewritten
A rewritten version of the [EarthMC Stats](https://github.com/EarthMC-Toolkit/EarthMC-Stats) Discord bot in Go.

## Why
I started this bot in hopes that it will be more powerful than EMCS with general perf and stability improvements.\
The database will be local, so you are responsible for upkeep of data. For 24/7 uptime, use a VPS or self-host with a Raspberry Pi etc.

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
>- `bot` -> Where the bot runs from. Contains all bot logic such as discord init, commands and events.
>- `utils` -> Provides small funcs generally re-used a lot like helpers for strings, slices, http etc.
>- `db` -> Where permanent data such as alliances are intended to be stored. Git ignored.
>- `api` -> Contains packages relating to APIs. Contains funcs that interact with both where necessary.
>     - `mapi` -> For interacting with the map API. (Currently Squaremap)
>     - `oapi` -> For interacting with the Official API.
