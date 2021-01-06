package discord

import (
	"fmt"
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
		_, err := instance.Session.ChannelMessageSend(message.ChannelID, msg)
		if err != nil {
			s.Log.WithError(err).Error("Unable to send custom command message.")
		}
		return true
	}
	return false
}
