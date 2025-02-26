# gw-bot

![do you like apples](https://media2.giphy.com/media/v1.Y2lkPTc5MGI3NjExdW54N2V3Mmw5M2U5MzJlazA0emV3bTl6eXZyaGJwbjhyMGVzb2pmMiZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/7M6Ih6SPNfAIg/giphy.gif)

Subscribe to [ShopGoodwill.com](https://shopgoodwill.com/) updates via Discord.

## Development

1. Set `DISCORD_TOKEN` env var.
2. Need a writeable volume to track subscriptions and updates in SQLite. By default `./gw-bot.db` is created.
3. Build: `go build`
4. Run: `./gw-bot` (or `./gw-bot -help` for options)
