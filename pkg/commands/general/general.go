package general

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

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

					fields = append(fields, newField)

				}
				embed.Fields = fields
				embeds = append(embeds, embed)
			}

			for _, embed := range embeds {
				instance.SendEmbedMessage(embed, message.ChannelID, "Unable to send commands embed.")
			}
		},
		RequiredPermissions: nil,
	})
	s.RegisterCommand(discord.Command{
		Name:                "8ball",
		HelpText:            "8ball gives you a response to your question.",
		Execute:             eightBall,
		RequiredPermissions: nil,
	})
	s.RegisterCommand(discord.Command{
		Name:                "flipcoin",
		HelpText:            "Flips a coin and gives you heads or tails.",
		Execute:             flipCoin,
		RequiredPermissions: []discord.Permission{discord.PermissionFlipCoin},
	})
	s.RegisterCommand(discord.Command{
		Name:                "rolldice",
		HelpText:            "Rolls two dice and gives you the sum.",
		Execute:             rollDice,
		RequiredPermissions: []discord.Permission{discord.PermissionRollDice},
	})
}

func eightBall(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	rand.Seed(time.Now().Unix())
	possibleResponses := []string{
		"Don’t count on it", "Outlook not so good", "My sources say no", "Very doubtful", "My reply is no", "It is certain",
		"Without a doubt", "You may rely on it", "Yes definitely", "It is decidedly so", "As I see it, yes", "Most likely",
		"Yes", "Outlook good", "Signs point to yes", "Reply hazy try again", "Better not tell you now", "Ask again later",
		"Cannot predict now", "Concentrate and ask again",
	}

	response := possibleResponses[rand.Intn(len(possibleResponses))]

	_, err := instance.Session.ChannelMessageSend(message.ChannelID,
		fmt.Sprintf("%s %s.", message.Author.Mention(), response))
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send channel message for 8ball.")
	}
}

func flipCoin(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	rand.Seed(time.Now().Unix())
	results := []string{
		"Tails",
		"Heads",
	}
	result := results[rand.Intn(len(results))]
	_, err := instance.Session.ChannelMessageSend(message.ChannelID,
		fmt.Sprintf("%s Flipped a coin and it landed on: %s", message.Author.Mention(), result))
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send channel message for flip coin.")
	}
}

func rollDice(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	rand.Seed(time.Now().Unix())
	result := rand.Intn(11) + 2
	_, err := instance.Session.ChannelMessageSend(message.ChannelID,
		fmt.Sprintf("%s Rolled two dice for a total of: %d out of 12", message.Author.Mention(), result))
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send channel message for flip coin.")
	}
}
