package moderation

import (
	"thalassa_discord/pkg/discord"
)

func RegisterCommands(s *discord.ShardInstance) {
	s.RegisterCommand(
		discord.Command{
			Name:                "mute",
			HelpText:            "",
			Execute:             muteUser,
			RequiredPermissions: []discord.Permission{discord.PermissionModerationMuteMember},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "unmute",
			HelpText:            "",
			Execute:             unmuteUser,
			RequiredPermissions: []discord.Permission{discord.PermissionModerationMuteMember},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "notifyme",
			HelpText:            "This will add you to the notify role. This means you want to be notified of announcements. Likely to be used instead of @everyone",
			Execute:             notifySubscribe,
			RequiredPermissions: nil,
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "unotifyme",
			HelpText:            "This will remove you from the notify role.",
			Execute:             notifyUnSubscribe,
			RequiredPermissions: nil,
		})
}
