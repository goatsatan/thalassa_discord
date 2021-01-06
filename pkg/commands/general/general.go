package general

import (
	"sort"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func RegisterCommands(s *discord.ShardInstance) {
	s.RegisterCommand(discord.Command{
		Name:     "commands",
		HelpText: "Displays all available commands.",
		Execute: func(instance *discord.ServerInstance, message *discordgo.Message, strings []string) {
			s.Log.Debug("Commands")
			var commandsArray []*discord.Command
			for _, command := range s.Commands {
				commandsArray = append(commandsArray, command)
			}

			sort.Slice(commandsArray, func(i, j int) bool {
				return commandsArray[i].Name < commandsArray[j].Name
			})

			var embedChunks [][]*discord.Command
			i := 0
			for i < len(commandsArray) {
				limit := 25
				if len(commandsArray) > i+limit {
					embedChunks = append(embedChunks, commandsArray[i:i+limit])
				} else {
					embedChunks = append(embedChunks, commandsArray[i:])
				}
				i += limit
			}

			var embeds []*discordgo.MessageEmbed

			for _, embedChunk := range embedChunks {
				embed := discord.NewEmbedInfer(instance.Session.State.User.Username, 28804).MessageEmbed
				embed.Title = "Commands"
				var fields []*discordgo.MessageEmbedField
				for _, command := range embedChunk {
					commandValue := ""
					if len(command.HelpText) > 250 {
						commandValue = command.HelpText[:247] + "..."
					} else {
						commandValue = command.HelpText
					}
					if commandValue == "" {
						commandValue = "No help text provided."
					}
					newField := &discordgo.MessageEmbedField{
						Name:   command.Name,
						Value:  commandValue,
						Inline: false,
					}

					// if i % 3 == 0 {
					// 	newField.Inline = false
					// }

					fields = append(fields, newField)

				}
				embed.Fields = fields
				embeds = append(embeds, embed)
			}

			for _, embed := range embeds {
				instance.SendEmbedMessage(embed, message.ChannelID, "Unable to send commands embed.")
			}

			// for z, chunk := range embedChunks {
			// 	s.Log.Debugf("Size of embed chunk %d: %d", z, len(chunk))
			// 	for _, c := range chunk {
			// 		s.Log.Debugf("Command in chunk %d: %s", z, c.Name)
			// 	}
			// }
		},
		RequiredPermissions: nil,
	})
}
