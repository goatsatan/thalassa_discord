package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (serverInstance *ServerInstance) GetOrCreateNotifyRole() (roleID string, err error) {
	var mutedRole *discordgo.Role
	foundNotifyRole := false
	guildRoles, err := serverInstance.Session.GuildRoles(serverInstance.GuildID)
	if err != nil {
		serverInstance.Log.Error().Err(err).Msg("Unable to get guild roles.")
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
		perms := int64(discordgo.PermissionReadMessageHistory | discordgo.PermissionViewChannel | discordgo.PermissionVoiceConnect)
		mentionable := false
		color := ORANGE
		newRole, err := serverInstance.Session.GuildRoleCreate(
			serverInstance.GuildID,
			&discordgo.RoleParams{
				Name:        "Notify Me",
				Color:       &color,
				Permissions: &perms,
				Mentionable: &mentionable,
			},
		)
		if err != nil {
			serverInstance.Log.Error().Err(err).Msg("Unable to create notify role.")
			return "", fmt.Errorf("unable to create notify role")
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
		serverInstance.Log.Error().Err(err).Msg("Unable to get guild roles.")
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
		color := 12370112
		perms := int64(discordgo.PermissionReadMessageHistory | discordgo.PermissionViewChannel | discordgo.PermissionVoiceConnect)
		mentionable := false
		newRole, err := serverInstance.Session.GuildRoleCreate(
			serverInstance.GuildID,
			&discordgo.RoleParams{
				Name:        "Muted",
				Color:       &color,
				Permissions: &perms,
				Mentionable: &mentionable,
			},
		)
		if err != nil {
			serverInstance.Log.Error().Err(err).Msg("Unable to create muted role.")
			return "", fmt.Errorf("unable to create muted role")
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
			serverInstance.Log.Error().Str("channel", channel.Name).Err(err).Msg("Unable to add muted role to channel.")
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
		serverInstance.Log.Error().Err(err).Msg("Unable to get guild channels.")
		return err
	}

	for _, channel := range guildChannels {
		_ = serverInstance.addMutedRoleToChannel(channel, mutedRoleID)
	}
	return nil
}
