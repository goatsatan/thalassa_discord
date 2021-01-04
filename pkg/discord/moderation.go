package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

func (serverInstance *ServerInstance) getOrCreateMutedRole() (roleID string, err error) {
	serverInstance.Lock()
	defer serverInstance.Unlock()
	var mutedRole *discordgo.Role
	foundMutedRole := false
	guildRoles, err := serverInstance.Session.GuildRoles(serverInstance.Guild.ID)
	if err != nil {
		serverInstance.Log.WithFields(logrus.Fields{
			"Guild": serverInstance.Guild.Name,
		}).WithError(err).Error("Unable to get guild roles.")
		return "", fmt.Errorf("unable to get guild roles")
	}
	for _, role := range guildRoles {
		if strings.ToLower(role.Name) == "muted" {
			mutedRole = role
			foundMutedRole = true
			break
		}
	}
	if !foundMutedRole {
		newRole, err := serverInstance.Session.GuildRoleCreate(serverInstance.Guild.ID)
		if err != nil {
			serverInstance.Log.WithField("Guild", serverInstance.Guild.Name).WithError(err).Error("Unable to create muted role.")
			return "", fmt.Errorf("unable to create muted role")
		}
		newRole, err = serverInstance.Session.GuildRoleEdit(
			serverInstance.Guild.ID,
			newRole.ID,
			"Muted",
			12370112,
			false,
			discordgo.PermissionReadMessageHistory|discordgo.PermissionViewChannel|discordgo.PermissionVoiceConnect,
			false,
		)
		if err != nil {
			serverInstance.Log.WithField("Guild", serverInstance.Guild.Name).WithError(err).Error("Unable to update muted role.")
			return "", err
		}
		mutedRole = newRole
	}
	if mutedRole == nil {
		return "", fmt.Errorf("muted role not found")
	}
	return mutedRole.ID, nil
}

func (serverInstance *ServerInstance) addMutedRoleToChannel(channel *discordgo.Channel, mutedRoleID string) error {
	foundMutedOverwrite := false
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.ID == mutedRoleID {
			foundMutedOverwrite = true
			break
		}
	}
	if !foundMutedOverwrite {
		err := serverInstance.Session.ChannelPermissionSet(channel.ID, mutedRoleID, "role", 0, 2553920)
		if err != nil {
			serverInstance.Log.WithFields(logrus.Fields{
				"Guild":   serverInstance.Guild.Name,
				"Channel": channel.Name,
			}).WithError(err).Error("Unable to add muted role to channel.")
		}
	}
	return nil
}

func (serverInstance *ServerInstance) addMutedRoleToAllChannels() error {
	mutedRoleID, err := serverInstance.getOrCreateMutedRole()
	if err != nil {
		return err
	}
	guildChannels, err := serverInstance.Session.GuildChannels(serverInstance.Guild.ID)
	if err != nil {
		serverInstance.Log.WithFields(logrus.Fields{
			"Guild": serverInstance.Guild.Name,
		}).WithError(err).Error("Unable to get guild channels.")
		return err
	}

	for _, channel := range guildChannels {
		_ = serverInstance.addMutedRoleToChannel(channel, mutedRoleID)
	}
	return nil
}
