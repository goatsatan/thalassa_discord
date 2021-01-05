package discord

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
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

func (serverInstance *ServerInstance) getUserPermissions(message *discordgo.Message) (*rolePermission, error) {
	commandMember, err := serverInstance.Session.State.Member(serverInstance.Guild.ID, message.Author.ID)
	if err != nil {
		commandMember, err = serverInstance.Session.GuildMember(serverInstance.Guild.ID, message.Author.ID)
		if err != nil {
			serverInstance.Log.WithFields(logrus.Fields{
				"Guild": serverInstance.Guild.Name,
				"User":  message.Author.Username,
			}).WithError(err).Error("Unable to get command member from message.")
			// TODO add error
			return nil, err
		}
	}

	userIsAdministrator := false
	for _, roleID := range commandMember.Roles {
		role, err := serverInstance.Session.State.Role(serverInstance.Guild.ID, roleID)
		if err != nil {
			serverInstance.Log.WithError(err).Error("Unable to get role permission.")
			break
		}
		if role.Permissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
			// User is has a role with administrator permissions.
			userIsAdministrator = true
			break
		}

		if serverInstance.Guild.OwnerID == message.Author.ID {
			// User is the owner of the server.
			userIsAdministrator = true
			break
		}
	}

	if userIsAdministrator {
		commandMemberPermissions := &rolePermission{
			roleID:                "User",
			postLinks:             true,
			moderationMuteMember:  true,
			rollDice:              true,
			flipCoin:              true,
			randomImage:           true,
			useCustomCommand:      true,
			manageCustomCommand:   true,
			ignoreCommandThrottle: true,
			playSongs:             true,
			playLists:             true,
			skipSongs:             true,
		}
		return commandMemberPermissions, nil
	}

	commandMemberPermissions := &rolePermission{
		roleID:                "User",
		postLinks:             false,
		moderationMuteMember:  false,
		rollDice:              false,
		flipCoin:              false,
		randomImage:           false,
		useCustomCommand:      false,
		manageCustomCommand:   false,
		ignoreCommandThrottle: false,
		playSongs:             false,
		playLists:             false,
		skipSongs:             false,
	}

	for _, r := range commandMember.Roles {
		for roleID, role := range serverInstance.rolePermissions {
			if r == roleID {
				switch {
				case role.postLinks:
					commandMemberPermissions.postLinks = true
				case role.moderationMuteMember:
					commandMemberPermissions.moderationMuteMember = true
				case role.rollDice:
					commandMemberPermissions.rollDice = true
				case role.flipCoin:
					commandMemberPermissions.flipCoin = true
				case role.randomImage:
					commandMemberPermissions.randomImage = true
				case role.useCustomCommand:
					commandMemberPermissions.useCustomCommand = true
				case role.manageCustomCommand:
					commandMemberPermissions.manageCustomCommand = true
				case role.ignoreCommandThrottle:
					commandMemberPermissions.ignoreCommandThrottle = true
				case role.playSongs:
					commandMemberPermissions.playSongs = true
				case role.playLists:
					commandMemberPermissions.playLists = true
				case role.skipSongs:
					commandMemberPermissions.skipSongs = true
				}
				break
			}
		}
	}
	return commandMemberPermissions, nil
}
