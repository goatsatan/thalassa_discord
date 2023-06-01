package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (serverInstance *ServerInstance) GetGuild() (*discordgo.Guild, error) {
	guild, err := serverInstance.Session.State.Guild(serverInstance.GuildID)
	if err != nil {
		guild, err = serverInstance.Session.Guild(serverInstance.GuildID)
		if err != nil {
			serverInstance.Log.Error().Err(err).Msg("Unable to get guild.")
			return nil, err
		}
	}
	return guild, nil
}

func (serverInstance *ServerInstance) GetGuildMember(userID string) (*discordgo.Member, error) {
	member, err := serverInstance.Session.State.Member(serverInstance.GuildID, userID)
	if err != nil {
		member, err = serverInstance.Session.GuildMember(serverInstance.GuildID, userID)
		if err != nil {
			serverInstance.Log.Error().Str("user_id", userID).Err(err).Msg("Unable to get member from guild.")
			return nil, err
		}
	}
	return member, err
}

func (serverInstance *ServerInstance) GetGuildRole(roleID string) (*discordgo.Role, error) {
	role, err := serverInstance.Session.State.Role(serverInstance.GuildID, roleID)
	if err != nil {
		roles, err := serverInstance.Session.GuildRoles(serverInstance.GuildID)
		if err != nil {
			serverInstance.Log.Error().Err(err).Msg("Unable to get guild role permissions.")
			return nil, err
		}
		for _, r := range roles {
			if r.ID == roleID {
				role = r
				break
			}
		}
	}
	return role, nil
}

func (serverInstance *ServerInstance) GetEveryoneRoleID() (string, error) {
	guild, err := serverInstance.GetGuild()
	if err != nil {
		return "", err
	}
	for _, role := range guild.Roles {
		if role.Name == "@everyone" {
			return role.ID, nil
		}
	}
	return "", fmt.Errorf("everyone role not found")
}
