package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

func (serverInstance *ServerInstance) GetGuild() (*discordgo.Guild, error) {
	guild, err := serverInstance.Session.State.Guild(serverInstance.GuildID)
	if err != nil {
		guild, err = serverInstance.Session.Guild(serverInstance.GuildID)
		if err != nil {
			serverInstance.Log.WithField("Guild ID", serverInstance.GuildID).WithError(err).Error("Unable to get guild.")
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
			serverInstance.Log.WithFields(logrus.Fields{
				"Guild ID": serverInstance.GuildID,
				"User ID":  userID,
			}).WithError(err).Error("Unable to get member from guild.")
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
			serverInstance.Log.WithError(err).Error("Unable to get role permission.")
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
