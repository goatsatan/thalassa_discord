package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (s *shardInstance) handleCustomCommand(commandName string, args []string, message *discordgo.Message, instance *ServerInstance) (foundCustom bool) {
	instance.RLock()
	customCommand, exists := instance.customCommands[commandName]
	instance.RUnlock()
	if exists {
		var msg string
		if len(args) != 0 {
			argsMsg := strings.Join(args, " ")
			msg = fmt.Sprintf("%s %s", argsMsg, customCommand)
		} else {
			msg = customCommand
		}
		_, err := instance.Session.ChannelMessageSend(message.ChannelID, msg)
		if err != nil {
			s.log.WithError(err).Error("Unable to send custom command message.")
		}
		return true
	}
	return false
}
