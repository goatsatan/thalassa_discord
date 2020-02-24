package main

import (
	"log"

	"thalassa_discord/pkg/discord"
)

func main() {
	discordInstance, err := discord.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	discordInstance.Start()
}
