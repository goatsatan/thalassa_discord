package music

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"thalassa_discord/models"
	"thalassa_discord/pkg/discord"
)

func next(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	nextSongs, errNext := models.SongRequests(
		qm.Where("guild_id = ?", instance.GuildID),
		qm.Where("played = false"),
		qm.OrderBy("requested_at ASC"),
		qm.Load(models.SongRequestRels.Song),
		qm.Limit(10),
	).All(instance.Ctx, instance.Db)
	if errNext != nil {
		instance.Log.Error().Err(errNext).Msg("Unable to query next songs in queue.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Unable to get next songs in queue.", "Database error.", false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, instance.Configuration.MusicTextChannelID.String, "Unable to send song count message.")
		return
	}
	embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, discord.GOLD).MessageEmbed
	embedmsg.Title = "Songs on the deck."
	embedmsg.Description = fmt.Sprintf("Showing the next %d songs in the queue.", len(nextSongs))
	switch len(nextSongs) {
	case 0:
		embedmsg.Description = "There are no songs in the queue."
	case 1:
		embedmsg.Description = "Showing the next song in the queue."
	}
	for index, song := range nextSongs {
		embedmsg.Fields = append(embedmsg.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%d.", index+1),
			Value:  fmt.Sprintf("[%s](%s)", song.SongName, song.R.Song.URL),
			Inline: false,
		})
	}
	instance.SendEmbedMessage(embedmsg, instance.Configuration.MusicTextChannelID.String, "Unable to send song count message.")
}
