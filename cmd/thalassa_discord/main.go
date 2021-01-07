package main

import (
	"log"

	"thalassa_discord/pkg/commands/example"
	"thalassa_discord/pkg/commands/general"
	"thalassa_discord/pkg/commands/lookup"
	"thalassa_discord/pkg/commands/moderation"
	"thalassa_discord/pkg/commands/music"
	"thalassa_discord/pkg/commands/random"
	"thalassa_discord/pkg/discord"
)

func main() {
	discordInstance, err := discord.NewInstance()
	if err != nil {
		log.Fatal(err)
	}

	// Register all created commands.
	moderation.RegisterCommands(discordInstance)
	lookup.RegisterCommands(discordInstance)
	music.RegisterCommands(discordInstance)
	random.RegisterCommands(discordInstance)
	example.RegisterCommands(discordInstance)
	general.RegisterCommands(discordInstance)

	discordInstance.Start()
}
