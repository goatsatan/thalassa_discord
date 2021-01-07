package example

import (
	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func RegisterCommands(s *discord.ShardInstance) {
	s.RegisterCommand(discord.Command{
		Name:     "example1",
		HelpText: "This is just an example command",
		Execute: func(instance *discord.ServerInstance, message *discordgo.Message, strings []string) {
			_, _ = instance.Session.ChannelMessageSend(message.ChannelID, "example 1")
		},
		RequiredPermissions: nil,
	})
	s.RegisterCommand(discord.Command{
		Name:                "example2",
		HelpText:            "This is just an example command",
		Execute:             example2,
		RequiredPermissions: nil,
	})
	// s.RegisterCommand(discord.Command{
	// 	FriendlyName:                "example3",
	// 	HelpText:            "This is just an example command",
	// 	Execute: func(instance *discord.ServerInstance, message *discordgo.Message, strings []string) {
	// 		// Example
	// 	},
	// 	RequiredPermissions: []discord.Permission{discord.PermissionFlipCoin},
	// })
}

func example2(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	_, _ = instance.Session.ChannelMessageSend(message.ChannelID, "example 2")
}
