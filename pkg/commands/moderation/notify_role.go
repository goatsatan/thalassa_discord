package moderation

import (
	"fmt"
	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func notifySubscribe(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	notifyEnabled := instance.Configuration.NotifyMeRoleEnabled
	instance.RUnlock()
	l := instance.Log.With().Fields(map[string]interface{}{
		"requester_username": message.Author.Username,
		"requester_id":       message.Author.ID,
		"channel_id":         message.ChannelID,
		"command":            "notify_subscribe",
	}).Logger()
	if notifyEnabled {
		notifyRoleID, err := instance.GetOrCreateNotifyRole()
		if err != nil {
			instance.SendErrorEmbed("Unable to get notify role.", err.Error(), message.ChannelID)
			return
		}
		err = instance.Session.GuildMemberRoleAdd(instance.GuildID, message.Author.ID, notifyRoleID)
		if err != nil {
			l.Error().Err(err).Msg("Unable to add notify role to user.")
			instance.SendErrorEmbed("Unable to add notify role to user", err.Error(), message.ChannelID)
			return
		}
	}
	_, err := instance.Session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Successfully added you to the notify role %s",
		message.Author.Mention()))
	if err != nil {
		l.Error().Err(err).Msg("Unable to send notify role added message.")
	}
}

func notifyUnSubscribe(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	notifyEnabled := instance.Configuration.NotifyMeRoleEnabled
	instance.RUnlock()
	l := instance.Log.With().Fields(map[string]interface{}{
		"requester_username": message.Author.Username,
		"requester_id":       message.Author.ID,
		"channel_id":         message.ChannelID,
		"command":            "notify_unsubscribe",
	}).Logger()
	if notifyEnabled {
		notifyRoleID, err := instance.GetOrCreateNotifyRole()
		if err != nil {
			instance.SendErrorEmbed("Unable to get notify role.", err.Error(), message.ChannelID)
			return
		}
		err = instance.Session.GuildMemberRoleRemove(instance.GuildID, message.Author.ID, notifyRoleID)
		if err != nil {
			l.Error().Err(err).Msg("Unable to remove notify role from user.")
			instance.SendErrorEmbed("Unable to remove notify role from user", err.Error(), message.ChannelID)
			return
		}
	}
	_, err := instance.Session.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Successfully removed the notify role from you %s",
		message.Author.Mention()))
	if err != nil {
		l.Error().Err(err).Msg("Unable to send notify role removed message.")
	}
}
