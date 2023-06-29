package main

import (
	"context"
	"log"

	"thalassa_discord/pkg/api"
	"thalassa_discord/pkg/commands/example"
	"thalassa_discord/pkg/commands/general"
	"thalassa_discord/pkg/commands/lookup"
	"thalassa_discord/pkg/commands/moderation"
	"thalassa_discord/pkg/commands/music"
	"thalassa_discord/pkg/commands/random"
	"thalassa_discord/pkg/discord"
)

func main() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	discordInstance, err := discord.NewInstance(ctx)
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

	if discordInstance.BotConfig.EnableAPI {
		apiInstance := api.New(discordInstance, discordInstance.BotConfig.APIHost, discordInstance.BotConfig.APIPort)
		discordInstance.SongQueueUpdateCallbackMutex.Lock()
		discordInstance.SongQueueUpdateCallback = apiInstance.SongQueueEventUpdate
		discordInstance.SongQueueUpdateCallbackMutex.Unlock()
		apiInstance.Start(discordInstance.Ctx)
	}

	discordInstance.Start()
}
