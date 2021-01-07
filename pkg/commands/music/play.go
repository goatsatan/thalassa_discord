package music

import (
	"context"
	"fmt"
	"time"

	"thalassa_discord/models"
	"thalassa_discord/pkg/discord"
	"thalassa_discord/pkg/music"
	"thalassa_discord/pkg/utils"

	"github.com/bwmarrin/discordgo"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func playSong(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	musicChatChannelID := instance.Configuration.MusicTextChannelID
	instance.RUnlock()
	err := instance.Session.ChannelMessageDelete(message.ChannelID, message.ID)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to delete message.")
	}
	if len(args) == 0 {
		_, err := instance.Session.ChannelMessageSend(message.ChannelID, "You must specify a URL to a song, video, or stream.")
		if err != nil {
			instance.Log.WithError(err).Error("Unable to send channel message.")
		}
		return
	}
	_, msgErr := instance.Session.ChannelMessageSend(message.ChannelID,
		fmt.Sprintf("🎵 Attempting to parse a song and add it to the queue that was requested by %s. 🎵",
			message.Author.Username))
	if msgErr != nil {
		instance.Log.WithError(msgErr).Error("Unable to send message about song queue addition.")
	}

	handleSongInfo(instance, message, musicChatChannelID.String, args[0])
	instance.MusicData.RLock()
	currentlyPlaying := instance.MusicData.SongPlaying
	instance.MusicData.RUnlock()
	if !currentlyPlaying {
		instance.HandleSong(musicChatChannelID.String)
	}
}

func handleSongInfo(instance *discord.ServerInstance, message *discordgo.Message, musicChatChannelID, url string) {
	// fmt.Println("Handling song info.")
	song, err := music.GetSongInfo(context.Background(), url)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get song info.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Error getting song information:", err.Error(), false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID, "Unable to send song info error message.")
		return
	}
	newSong := &models.Song{
		ID:                song.ID,
		Platform:          null.StringFrom(song.ExtractorKey),
		SongName:          song.Title,
		Description:       null.StringFrom(song.Description),
		URL:               url,
		DurationInSeconds: null.IntFrom(song.Duration),
		IsStream:          false,
		ThumbnailURL:      null.StringFrom(song.Thumbnail),
		Artist:            utils.InterfaceToNullString(song.Artist),
		Album:             utils.InterfaceToNullString(song.Album),
		Track:             utils.InterfaceToNullString(song.Track),
	}
	err = newSong.Upsert(context.Background(), instance.Db, true, []string{"id"}, boil.Infer(), boil.Infer())
	if err != nil {
		instance.Log.WithError(err).Error("Unable to upsert song.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Unable to add song request.", "Database error.", false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID, "Unable to upsert song.")
		return
	}
	newSongRequest := &models.SongRequest{
		SongID:            null.StringFrom(newSong.ID),
		SongName:          song.Title,
		RequestedByUserID: message.Author.ID,
		UsernameAtTime:    message.Author.Username,
		GuildID:           message.GuildID,
		GuildNameAtTime:   instance.GuildID,
		RequestedAt:       time.Now().UTC(),
		PlayedAt:          null.Time{},
		Played:            false,
	}
	err = newSongRequest.Insert(context.Background(), instance.Db, boil.Infer())
	if err != nil {
		instance.Log.WithError(err).Error("Unable to insert song request.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Unable to add song request.", "Database error.", false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID, "Unable to insert song request.")
		return
	}
	embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 28804).
		AddField("Song has been added to the queue", song.Title, false).
		AddField("Requested By", message.Author.Username, false).
		MessageEmbed
	instance.SendEmbedMessage(embedmsg, musicChatChannelID, "Unable to send song added to queue message.")
	return
}
