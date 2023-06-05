package music

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"thalassa_discord/models"
	"thalassa_discord/pkg/discord"
)

func songCount(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	count, errCount := models.SongRequests(
		qm.Where("guild_id = ?", instance.GuildID),
		qm.Where("played = false"),
	).Count(instance.Ctx, instance.Db)
	if errCount != nil {
		instance.Log.Error().Err(errCount).Msg("Unable to query song request count.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xff9999).
			AddField("Unable to get number of songs in queue.", "Database error.", false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, instance.Configuration.MusicTextChannelID.String, "Unable to send song count message.")
		return
	}
	embedmsg := discord.NewEmbedInfer(instance.Session.State.User, discord.DARK_GREEN).
		AddField("Number of songs in queue", strconv.FormatInt(count, 10), false).
		MessageEmbed
	instance.SendEmbedMessage(embedmsg, instance.Configuration.MusicTextChannelID.String, "Unable to send song count message.")
}
