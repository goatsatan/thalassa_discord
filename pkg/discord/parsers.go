package discord

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (s *shardInstance) parseMessageForCommand(message *discordgo.Message, instance *ServerInstance,
) (foundCommand bool, command string, arguments []string) {
	if len(message.Content) > 0 {
		instance.RLock()
		prefix := instance.Configuration.PrefixCommand
		instance.RUnlock()
		if string(message.Content[0]) == prefix {
			splitMsg := strings.Split(message.Content, " ")
			commandName := splitMsg[0]
			commandName = strings.ToLower(commandName)
			commandName = commandName[1:]
			args := splitMsg[1:]
			return true, commandName, args
		}
	}
	return false, "", nil
}

func getDiscordUserIDFromString(mention string) (string, bool) {
	re := regexp.MustCompile("<@(.+)>")
	s := re.FindStringSubmatch(mention)
	re = regexp.MustCompile("!")
	if len(s) >= 2 {
		userID := re.ReplaceAllString(s[1], "")
		return userID, true
	} else {
		return "", false
	}
}
