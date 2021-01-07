package random

import (
	"thalassa_discord/pkg/discord"
)

func RegisterCommands(s *discord.ShardInstance) {
	s.RegisterCommand(
		discord.Command{
			Name:                "dog",
			HelpText:            "This gets a random picture of a dog.",
			Execute:             getRandomDogPicture,
			RequiredPermissions: []discord.Permission{discord.PermissionRandomImage},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "cat",
			HelpText:            "This gets a random picture of a cat.",
			Execute:             getRandomCatPicture,
			RequiredPermissions: []discord.Permission{discord.PermissionRandomImage},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "fox",
			HelpText:            "This gets a random picture of a fox.",
			Execute:             getRandomFoxPicture,
			RequiredPermissions: []discord.Permission{discord.PermissionRandomImage},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "joke",
			HelpText:            "This gets a random joke. If you're easily offended use safejoke instead.",
			Execute:             getRandomJoke,
			RequiredPermissions: nil,
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "safejoke",
			HelpText:            "This gets a random joke.",
			Execute:             getRandomSafeJoke,
			RequiredPermissions: nil,
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "duck",
			HelpText:            "This gets a random picture of a duck.",
			Execute:             getRandomDuckPicture,
			RequiredPermissions: []discord.Permission{discord.PermissionRandomImage},
		})
	s.RegisterCommand(
		discord.Command{
			Name:                "shibe",
			HelpText:            "This gets a random picture of a shibe.",
			Execute:             getRandomShibePicture,
			RequiredPermissions: []discord.Permission{discord.PermissionRandomImage},
		})
}
