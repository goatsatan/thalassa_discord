package music

import (
	"thalassa_discord/pkg/discord"
)

func RegisterCommands(s *discord.ShardInstance) {
	s.RegisterCommand(
		discord.Command{
			Name:                "play",
			HelpText:            "This command takes a URL and tries to play the audio in voice chat.",
			Execute:             playSong,
			RequiredPermissions: []discord.Permission{discord.PermissionPlaySongs},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "skip",
			HelpText:            "This skips the current playing song.",
			Execute:             skipSong,
			RequiredPermissions: []discord.Permission{discord.PermissionSkipSongs},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "skipall",
			HelpText:            "This skips all songs in the queue as well as the current playing song.",
			Execute:             skipAllSongs,
			RequiredPermissions: []discord.Permission{discord.PermissionSkipSongs},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "playlist",
			HelpText:            "This skips all songs in the queue as well as the current playing song.",
			Execute:             playList,
			RequiredPermissions: []discord.Permission{discord.PermissionPlayLists},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "songcount",
			HelpText:            "This command returns the number of songs in the queue.",
			Execute:             songCount,
			RequiredPermissions: nil,
		})
}
