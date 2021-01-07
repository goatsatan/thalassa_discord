package discord

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

//goland:noinspection GoUnusedConst
const (
	AQUA                = 1752220
	GREEN               = 3066993
	BLUE                = 3447003
	PURPLE              = 10181046
	GOLD                = 15844367
	ORANGE              = 15105570
	RED                 = 15158332
	GREY                = 9807270
	DARKER_GREY         = 8359053
	NAVY                = 3426654
	DARK_AQUA           = 1146986
	DARK_GREEN          = 2067276
	DARK_BLUE           = 2123412
	DARK_PURPLE         = 7419530
	DARK_GOLD           = 12745742
	DARK_ORANGE         = 11027200
	DARK_RED            = 10038562
	DARK_GREY           = 9936031
	LIGHT_GREY          = 12370112
	DARK_NAVY           = 2899536
	LUMINOUS_VIVID_PINK = 16580705
	DARK_VIVID_PINK     = 12320855
)

// TODO
// Add context to music. Deadline and cancel
// Add web portal to configure bot
// Add channel clean up
// Add channel clean up per user
// Add custom middleware to read all messages for keywords and auto respond
// Add expiration to muted users

// Command is the basic structure for every built-in and add-on command for the bot. It takes a name
// (this is used with the prefix to run the command). Help text is displayed if the user requests help for that command.
// execute is the function that gets run.
// required permissions gets run against the permission middleware. If you want to restrict the usage to certain roles
// before your command is run apply them. If no permissions are needed or you want to do the permission check yourself
// just use nil here.
type Command struct {
	Name                string
	HelpText            string
	Execute             func(*ServerInstance, *discordgo.Message, []string)
	RequiredPermissions []Permission
}

// RegisterCommand registers a new valid command for the bot to use globally. You must provide a full command.
func (s *ShardInstance) RegisterCommand(command Command) {
	s.Lock()
	s.Commands[strings.ToLower(command.Name)] = &command
	s.Unlock()
}

// handleCommand runs every time a command prefix is found in a message.
func (s *ShardInstance) handleCommand(commandName string, args []string, message *discordgo.Message, instance *ServerInstance) {
	command, exists := s.Commands[commandName]
	if exists {
		if command.userHasCommandPermission(instance, message) {
			command.Execute(instance, message, args)
		}
	}
}

// userHasCommandPermission is a middleware function that gets run before every command. It simply ignores the command if the user
// doesn't have permission.
func (c Command) userHasCommandPermission(instance *ServerInstance, message *discordgo.Message) bool {
	userPerms, err := instance.getUserPermissions(message)
	if err != nil {
		return false
	}

	// instance.Log.WithField("User", message.Author.Username).Debugf("%#v\n", userPerms)

	// Check for all permissions.
	_, exists := userPerms[PermissionAdministrator]
	if exists {
		return true
	}

	// Loop through every required permission. If it doesn't exist in the user permissions then we return false.
	for _, perm := range c.RequiredPermissions {
		_, exists := userPerms[perm]
		if !exists {
			return false
		}
	}
	return true
}
