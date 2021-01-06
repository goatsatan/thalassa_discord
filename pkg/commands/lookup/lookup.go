package lookup

import (
	"thalassa_discord/pkg/discord"
)

func RegisterCommands(s *discord.ShardInstance) {
	s.RegisterCommand(
		discord.Command{
			Name:                "define",
			HelpText:            "This gets the definition for a word.",
			Execute:             getDefinition,
			RequiredPermissions: nil,
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "iplookup",
			HelpText:            "This gets IP address information",
			Execute:             ipLookup,
			RequiredPermissions: nil,
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "udefine",
			HelpText:            "Urban dictionary lookup.",
			Execute:             urbanDictionary,
			RequiredPermissions: nil,
		})
}
