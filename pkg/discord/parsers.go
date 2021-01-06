package discord

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Permission int

const (
	PermissionPostLinks Permission = iota
	PermissionModerationMuteMember
	PermissionRollDice
	PermissionFlipCoin
	PermissionRandomImage
	PermissionUseCustomCommand
	PermissionManageCustomCommand
	PermissionIgnoreCommandThrottle
	PermissionPlaySongs
	PermissionPlayLists
	PermissionSkipSongs
	PermissionAll
)

func (s *ShardInstance) parseMessageForCommand(message *discordgo.Message, instance *ServerInstance,
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

func GetDiscordUserIDFromString(mention string) (string, bool) {
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

func (serverInstance *ServerInstance) getUserPermissions(message *discordgo.Message) (map[Permission]struct{}, error) {
	commandMember, err := serverInstance.GetGuildMember(message.Author.ID)
	if err != nil {
		return nil, err
	}

	userIsAdministrator := false
	for _, roleID := range commandMember.Roles {
		role, err := serverInstance.GetGuildRole(roleID)
		if err != nil {
			serverInstance.Log.WithError(err).Error("Unable to get role permission.")
			break
		}
		if role.Permissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
			// User is has a role with administrator permissions.
			userIsAdministrator = true
			break
		}

		guild, err := serverInstance.GetGuild()
		if err == nil {
			if guild.OwnerID == message.Author.ID {
				// User is the owner of the server.
				userIsAdministrator = true
				break
			}
		}

	}

	perms := make(map[Permission]struct{})

	if userIsAdministrator {
		perms[PermissionAll] = struct{}{}
		return perms, nil
	}

	// if userIsAdministrator {
	// 	commandMemberPermissions := &rolePermission{
	// 		roleID:                "User",
	// 		postLinks:             true,
	// 		moderationMuteMember:  true,
	// 		rollDice:              true,
	// 		flipCoin:              true,
	// 		randomImage:           true,
	// 		useCustomCommand:      true,
	// 		manageCustomCommand:   true,
	// 		ignoreCommandThrottle: true,
	// 		playSongs:             true,
	// 		playLists:             true,
	// 		skipSongs:             true,
	// 	}
	// 	return commandMemberPermissions, nil
	// }
	//
	// commandMemberPermissions := &rolePermission{
	// 	roleID:                "User",
	// 	postLinks:             false,
	// 	moderationMuteMember:  false,
	// 	rollDice:              false,
	// 	flipCoin:              false,
	// 	randomImage:           false,
	// 	useCustomCommand:      false,
	// 	manageCustomCommand:   false,
	// 	ignoreCommandThrottle: false,
	// 	playSongs:             false,
	// 	playLists:             false,
	// 	skipSongs:             false,
	// }

	// for _, r := range commandMember.Roles {
	// 	for roleID, role := range serverInstance.rolePermissions {
	// 		if r == roleID {
	// 			switch {
	// 			case role.postLinks:
	// 				commandMemberPermissions.postLinks = true
	// 			case role.moderationMuteMember:
	// 				commandMemberPermissions.moderationMuteMember = true
	// 			case role.rollDice:
	// 				commandMemberPermissions.rollDice = true
	// 			case role.flipCoin:
	// 				commandMemberPermissions.flipCoin = true
	// 			case role.randomImage:
	// 				commandMemberPermissions.randomImage = true
	// 			case role.useCustomCommand:
	// 				commandMemberPermissions.useCustomCommand = true
	// 			case role.manageCustomCommand:
	// 				commandMemberPermissions.manageCustomCommand = true
	// 			case role.ignoreCommandThrottle:
	// 				commandMemberPermissions.ignoreCommandThrottle = true
	// 			case role.playSongs:
	// 				commandMemberPermissions.playSongs = true
	// 			case role.playLists:
	// 				commandMemberPermissions.playLists = true
	// 			case role.skipSongs:
	// 				commandMemberPermissions.skipSongs = true
	// 			}
	// 			break
	// 		}
	// 	}
	// }
	// return commandMemberPermissions, nil

	userRoles := commandMember.Roles

	// Add @everyone role to user roles since it's not included by default.
	everyoneGroupID, err := serverInstance.GetEveryoneRoleID()
	if err == nil {
		userRoles = append(userRoles, everyoneGroupID)
	}

	for _, r := range commandMember.Roles {
		for roleID, role := range serverInstance.rolePermissions {
			if r == roleID {
				switch {
				case role.postLinks:
					perms[PermissionPostLinks] = struct{}{}
				case role.moderationMuteMember:
					perms[PermissionModerationMuteMember] = struct{}{}
				case role.rollDice:
					perms[PermissionRollDice] = struct{}{}
				case role.flipCoin:
					perms[PermissionFlipCoin] = struct{}{}
				case role.randomImage:
					perms[PermissionRandomImage] = struct{}{}
				case role.useCustomCommand:
					perms[PermissionUseCustomCommand] = struct{}{}
				case role.manageCustomCommand:
					perms[PermissionManageCustomCommand] = struct{}{}
				case role.ignoreCommandThrottle:
					perms[PermissionIgnoreCommandThrottle] = struct{}{}
				case role.playSongs:
					perms[PermissionPlaySongs] = struct{}{}
				case role.playLists:
					perms[PermissionPlayLists] = struct{}{}
				case role.skipSongs:
					perms[PermissionSkipSongs] = struct{}{}
				}
				break
			}
		}
	}
	return perms, nil
}
