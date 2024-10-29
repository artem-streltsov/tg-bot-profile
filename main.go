package main

import (
	"tg-bot-profile/bot"
	"tg-bot-profile/config"
	"tg-bot-profile/database"
)

func main() {
	cfg := config.LoadConfig()
	database.InitDB()
	bot.StartBot(cfg)
}
