package discord

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (s *ShardInstance) handleCustomCommand(commandName string, args []string, message *discordgo.Message, instance *ServerInstance) (foundCustom bool) {
	instance.RLock()
	customCommand, exists := instance.CustomCommands[commandName]
	instance.RUnlock()
	if exists {
		var msg string
		if len(args) != 0 {
			argsMsg := strings.Join(args, " ")
			msg = fmt.Sprintf("%s %s", argsMsg, customCommand)
		} else {
			msg = customCommand
		}
		if strings.Contains(msg, `\n`) {
			msg = strings.ReplaceAll(msg, `\n`, `
`)
		}
		_, err := instance.Session.ChannelMessageSend(message.ChannelID, msg)
		if err != nil {
			log.Error().Err(err).Msg("Unable to send custom command message.")
		}
		return true
	}
	return false
}
