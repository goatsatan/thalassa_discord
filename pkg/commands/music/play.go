package music

import (
	"fmt"
	"math"
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
		instance.Log.Error().Err(err).Msg("Unable to delete message.")
	}
	if len(args) == 0 {
		_, errMessageSend := instance.Session.ChannelMessageSend(message.ChannelID, "You must specify a URL to a song, video, or stream.")
		if errMessageSend != nil {
			instance.Log.Error().Err(errMessageSend).Msg("Unable to send channel message.")
		}
		return
	}
	// Send a message to the channel saying we're attempting to parse the song.
	embedmsg := discord.NewEmbedInfer(instance.Session.State.User, discord.DARKER_GREY).
		SetTitle("Attempting to parse a song and add it to the queue").
		AddField("Requested by", message.Author.Username, false).
		SetThumbnail("https://img.icons8.com/arcade/64/playlist.png").
		MessageEmbed
	go instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send message about song queue addition.")

	songInfo, err := music.GetSongInfo(instance.Ctx, args[0])
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get song info.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xff9999).
			AddField("Error getting song information:", err.Error(), false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send song info error message.")
		return
	}

	handleSongInfo(instance, message, musicChatChannelID.String, songInfo)
	instance.MusicData.RLock()
	currentlyPlaying := instance.MusicData.SongPlaying
	instance.MusicData.RUnlock()
	if !currentlyPlaying {
		instance.HandleSong(musicChatChannelID.String)
	}
}

func handleSongInfo(instance *discord.ServerInstance, message *discordgo.Message, musicChatChannelID string, songInfo *music.Song) {
	// fmt.Println("Handling song info.")
	if songInfo.Thumbnail == "" && len(songInfo.Thumbnails) > 0 {
		songInfo.Thumbnail = songInfo.Thumbnails[len(songInfo.Thumbnails)-1].URL
	}
	newSong := &models.Song{
		ID:                songInfo.ID,
		Platform:          null.StringFrom(songInfo.ExtractorKey),
		SongName:          songInfo.Title,
		Description:       null.StringFrom(songInfo.Description),
		URL:               songInfo.WebpageURL,
		DurationInSeconds: null.IntFrom(int(math.Round(songInfo.Duration))),
		IsStream:          false,
		ThumbnailURL:      null.StringFrom(songInfo.Thumbnail),
		Artist:            utils.InterfaceToNullString(songInfo.Artist),
		Album:             utils.InterfaceToNullString(songInfo.Album),
		Track:             utils.InterfaceToNullString(songInfo.Track),
	}
	errUpsertSong := newSong.Upsert(instance.Ctx, instance.Db, true, []string{"id"}, boil.Infer(), boil.Infer())
	if errUpsertSong != nil {
		instance.Log.Error().Err(errUpsertSong).Msg("Unable to upsert song.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xff9999).
			AddField("Unable to add song request.", "Database error.", false).
			MessageEmbed
		go instance.SendEmbedMessage(embedmsg, musicChatChannelID, "Unable to upsert song.")
		return
	}
	newSongRequest := &models.SongRequest{
		SongID:            null.StringFrom(newSong.ID),
		SongName:          songInfo.Title,
		RequestedByUserID: message.Author.ID,
		UsernameAtTime:    message.Author.Username,
		GuildID:           message.GuildID,
		GuildNameAtTime:   instance.GuildID,
		RequestedAt:       time.Now().UTC(),
		PlayedAt:          null.Time{},
		Played:            false,
	}
	errUpsertSongRequest := newSongRequest.Insert(instance.Ctx, instance.Db, boil.Infer())
	if errUpsertSongRequest != nil {
		instance.Log.Error().Err(errUpsertSongRequest).Msg("Unable to insert song request.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xff9999).
			AddField("Unable to add song request.", "Database error.", false).
			MessageEmbed
		go instance.SendEmbedMessage(embedmsg, musicChatChannelID, "Unable to insert song request.")
		return
	}
	embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 28804).
		AddField("Song has been added to the queue", fmt.Sprintf("[%s](%s)",
			songInfo.Title, songInfo.WebpageURL), false).
		AddField("Requested By", message.Author.Username, false).
		SetImage(songInfo.Thumbnail).
		MessageEmbed
	go instance.SendEmbedMessage(embedmsg, musicChatChannelID, "Unable to send song added to queue message.")
	return
}
