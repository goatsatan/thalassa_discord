package moderation

import (
	"context"
	"time"

	"thalassa_discord/models"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func muteUser(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	if len(args) > 0 {
		userID, found := discord.GetDiscordUserIDFromString(args[0])
		if !found {
			instance.SendErrorEmbed("Invalid command argument.", "Unable to find user from specified id.", message.ChannelID)
			return
		}

		l := instance.Log.With().Fields(map[string]interface{}{
			"requester_username": message.Author.Username,
			"requester_id":       message.Author.ID,
			"muted_user_id":      userID,
		}).Logger()

		err := instance.MuteUser(userID)
		if err != nil {
			instance.SendErrorEmbed("Unable to mute user", err.Error(), message.ChannelID)
			return
		}

		mutedModel := models.MutedMember{
			UserID:    userID,
			GuildID:   instance.GuildID,
			CreatedAt: time.Now(),
			ExpiresAt: null.Time{},
		}
		err = mutedModel.Upsert(context.TODO(), instance.Db, true, []string{"user_id", "guild_id"},
			boil.Whitelist("created_at", "expires_at"), boil.Infer())

		if err != nil {
			l.Error().Err(err).Msg("Unable to add muted user to database.")
		}

		guild, err := instance.GetGuild()
		if err != nil {
			l.Error().Err(err).Msg("Unable to get guild.")
		}

		if guild != nil {
			for _, vu := range guild.VoiceStates {
				if vu.UserID == userID {
					err := instance.Session.GuildMemberMove(instance.GuildID, userID, nil)
					if err != nil {
						l.Error().Err(err).Msg("Unable to disconnect muted member from voice.")
						instance.SendErrorEmbed("Unable to disconnect user from voice to mute.", "You must disconnect them manually.", message.ChannelID)
					}
				}
			}
		}

		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 28804).
			AddField("Successfully Muted User", args[0], false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, message.ChannelID, "Unable to send muted user message.")
	} else {
		instance.SendErrorEmbed("Invalid command argument.", "You must specify a user.", message.ChannelID)
	}
}

func unmuteUser(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	mutedRoleID, err := instance.GetOrCreateMutedRole()
	if err != nil {
		instance.SendErrorEmbed("Unable to get muted role.", err.Error(), message.ChannelID)
		return
	}
	if len(args) > 0 {
		userID, found := discord.GetDiscordUserIDFromString(args[0])
		if !found {
			instance.SendErrorEmbed("Invalid command argument.", "Unable to find user from specified id.", message.ChannelID)
			return
		}
		l := instance.Log.With().Fields(map[string]interface{}{
			"requester_username": message.Author.Username,
			"requester_id":       message.Author.ID,
			"muted_user_id":      userID,
		}).Logger()
		err := instance.Session.GuildMemberRoleRemove(instance.GuildID, userID, mutedRoleID)
		if err != nil {
			l.Error().Err(err).Msg("Unable to remove muted role from user.")
			instance.SendErrorEmbed("Unable to remove muted role from user.", err.Error(), message.ChannelID)
			return
		}

		mutedModel, err := models.MutedMembers(
			qm.Where("user_id = ?", userID),
			qm.And("guild_id = ?", instance.GuildID),
		).One(context.TODO(), instance.Db)
		if err != nil {
			l.Error().Err(err).Msg("Unable to get muted member from database")
		} else {
			_, err := mutedModel.Delete(context.TODO(), instance.Db)
			if err != nil {
				l.Error().Err(err).Msg("Unable to remove muted user from database.")
				instance.SendErrorEmbed("Database error.", "Unable to remove muted member.", message.ChannelID)
			}
		}

		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 28804).
			AddField("Successfully Unmuted User", args[0], false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, message.ChannelID, "Unable to send unmuted user message.")
	} else {
		instance.SendErrorEmbed("Invalid command argument.", "You must specify a user.", message.ChannelID)
	}
}
