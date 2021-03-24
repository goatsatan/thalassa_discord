package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

func (serverInstance *ServerInstance) GetOrCreateNotifyRole() (roleID string, err error) {
	var mutedRole *discordgo.Role
	foundNotifyRole := false
	guildRoles, err := serverInstance.Session.GuildRoles(serverInstance.GuildID)
	if err != nil {
		serverInstance.Log.WithFields(logrus.Fields{
			"Guild": serverInstance.GuildID,
		}).WithError(err).Error("Unable to get guild roles.")
		return "", fmt.Errorf("unable to get guild roles")
	}
	for _, role := range guildRoles {
		if strings.ToLower(role.Name) == "notify me" {
			mutedRole = role
			foundNotifyRole = true
			break
		}
	}
	if !foundNotifyRole {
		newRole, err := serverInstance.Session.GuildRoleCreate(serverInstance.GuildID)
		if err != nil {
			serverInstance.Log.WithField("Guild", serverInstance.GuildID).WithError(err).Error("Unable to create notify role.")
			return "", fmt.Errorf("unable to create notify role")
		}
		newRole, err = serverInstance.Session.GuildRoleEdit(
			serverInstance.GuildID,
			newRole.ID,
			"Notify Me",
			ORANGE,
			false,
			discordgo.PermissionReadMessageHistory|discordgo.PermissionViewChannel|discordgo.PermissionVoiceConnect,
			false,
		)
		if err != nil {
			serverInstance.Log.WithField("Guild", serverInstance.GuildID).WithError(err).Error("Unable to update notify role.")
			return "", err
		}
		mutedRole = newRole
	}
	if mutedRole == nil {
		return "", fmt.Errorf("notify role not found")
	}
	return mutedRole.ID, nil
}

func (serverInstance *ServerInstance) GetOrCreateMutedRole() (roleID string, err error) {
	var mutedRole *discordgo.Role
	foundMutedRole := false
	guildRoles, err := serverInstance.Session.GuildRoles(serverInstance.GuildID)
	if err != nil {
		serverInstance.Log.WithFields(logrus.Fields{
			"Guild": serverInstance.GuildID,
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
		newRole, err := serverInstance.Session.GuildRoleCreate(serverInstance.GuildID)
		if err != nil {
			serverInstance.Log.WithField("Guild", serverInstance.GuildID).WithError(err).Error("Unable to create muted role.")
			return "", fmt.Errorf("unable to create muted role")
		}
		newRole, err = serverInstance.Session.GuildRoleEdit(
			serverInstance.GuildID,
			newRole.ID,
			"Muted",
			12370112,
			false,
			discordgo.PermissionReadMessageHistory|discordgo.PermissionViewChannel|discordgo.PermissionVoiceConnect,
			false,
		)
		if err != nil {
			serverInstance.Log.WithField("Guild", serverInstance.GuildID).WithError(err).Error("Unable to update muted role.")
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
		err := serverInstance.Session.ChannelPermissionSet(channel.ID, mutedRoleID,
			discordgo.PermissionOverwriteTypeRole, 0, 2553920)
		if err != nil {
			serverInstance.Log.WithFields(logrus.Fields{
				"Guild":   serverInstance.GuildID,
				"Channel": channel.Name,
			}).WithError(err).Error("Unable to add muted role to channel.")
		}
	}
	return nil
}

func (serverInstance *ServerInstance) addMutedRoleToAllChannels() error {
	mutedRoleID, err := serverInstance.GetOrCreateMutedRole()
	if err != nil {
		return err
	}
	guildChannels, err := serverInstance.Session.GuildChannels(serverInstance.GuildID)
	if err != nil {
		serverInstance.Log.WithFields(logrus.Fields{
			"Guild": serverInstance.GuildID,
		}).WithError(err).Error("Unable to get guild channels.")
		return err
	}

	for _, channel := range guildChannels {
		_ = serverInstance.addMutedRoleToChannel(channel, mutedRoleID)
	}
	return nil
}
