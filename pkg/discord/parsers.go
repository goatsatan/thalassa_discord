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
	PermissionAdministrator
	PermissionOwner
)

var AllPermissions = []Permission{
	PermissionPostLinks,
	PermissionModerationMuteMember,
	PermissionRollDice,
	PermissionFlipCoin,
	PermissionRandomImage,
	PermissionUseCustomCommand,
	PermissionManageCustomCommand,
	PermissionIgnoreCommandThrottle,
	PermissionPlaySongs,
	PermissionPlayLists,
	PermissionSkipSongs,
	PermissionAdministrator,
	PermissionOwner,
}

func (p Permission) FriendlyName() string {
	switch p {
	case PermissionPostLinks:
		return "Post Links"
	case PermissionModerationMuteMember:
		return "Moderation Mute Member"
	case PermissionRollDice:
		return "Roll Dice"
	case PermissionFlipCoin:
		return "Flip Coin"
	case PermissionRandomImage:
		return "Random Image"
	case PermissionUseCustomCommand:
		return "Use Custom Command"
	case PermissionManageCustomCommand:
		return "Manage Custom Command"
	case PermissionIgnoreCommandThrottle:
		return "Ignore Command Throttle"
	case PermissionPlaySongs:
		return "Play Songs"
	case PermissionPlayLists:
		return "Play Lists"
	case PermissionSkipSongs:
		return "Skip Songs"
	case PermissionAdministrator:
		return "Administrator"
	case PermissionOwner:
		return "Owner"
	default:
		return "Unknown"
	}
}

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

	perms := make(map[Permission]struct{})

	for _, roleID := range commandMember.Roles {
		role, err := serverInstance.GetGuildRole(roleID)
		if err != nil {
			serverInstance.Log.Error().Err(err).Msg("Unable to get role permission.")
			break
		}
		if role.Permissions&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
			// User is has a role with administrator permissions.
			perms[PermissionAdministrator] = struct{}{}
			return perms, nil
		}

		guild, err := serverInstance.GetGuild()
		if err == nil {
			if guild.OwnerID == message.Author.ID {
				// User is the owner of the server.
				perms[PermissionAdministrator] = struct{}{}
				perms[PermissionOwner] = struct{}{}
				return perms, nil
			}
		}

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
	// 			if role.postLinks:
	// 				commandMemberPermissions.postLinks = true
	// 			if role.moderationMuteMember:
	// 				commandMemberPermissions.moderationMuteMember = true
	// 			if role.rollDice:
	// 				commandMemberPermissions.rollDice = true
	// 			if role.flipCoin:
	// 				commandMemberPermissions.flipCoin = true
	// 			if role.randomImage:
	// 				commandMemberPermissions.randomImage = true
	// 			if role.useCustomCommand:
	// 				commandMemberPermissions.useCustomCommand = true
	// 			if role.manageCustomCommand:
	// 				commandMemberPermissions.manageCustomCommand = true
	// 			if role.ignoreCommandThrottle:
	// 				commandMemberPermissions.ignoreCommandThrottle = true
	// 			if role.playSongs:
	// 				commandMemberPermissions.playSongs = true
	// 			if role.playLists:
	// 				commandMemberPermissions.playLists = true
	// 			if role.skipSongs:
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

	for _, r := range userRoles {
		for roleID, role := range serverInstance.rolePermissions {
			if r == roleID {
				if role.postLinks {
					perms[PermissionPostLinks] = struct{}{}
				}
				if role.moderationMuteMember {
					perms[PermissionModerationMuteMember] = struct{}{}
				}
				if role.rollDice {
					perms[PermissionRollDice] = struct{}{}
				}
				if role.flipCoin {
					perms[PermissionFlipCoin] = struct{}{}
				}
				if role.randomImage {
					perms[PermissionRandomImage] = struct{}{}
				}
				if role.useCustomCommand {
					perms[PermissionUseCustomCommand] = struct{}{}
				}
				if role.manageCustomCommand {
					perms[PermissionManageCustomCommand] = struct{}{}
				}
				if role.ignoreCommandThrottle {
					perms[PermissionIgnoreCommandThrottle] = struct{}{}
				}
				if role.playSongs {
					perms[PermissionPlaySongs] = struct{}{}
				}
				if role.playLists {
					perms[PermissionPlayLists] = struct{}{}
				}
				if role.skipSongs {
					perms[PermissionSkipSongs] = struct{}{}
				}
			}
		}
	}
	return perms, nil
}
