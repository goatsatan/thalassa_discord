package music

import (
	"context"
	"fmt"

	"thalassa_discord/pkg/discord"
	"thalassa_discord/pkg/music"

	"github.com/bwmarrin/discordgo"
)

func skipSong(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	songPlaying := instance.MusicData.SongPlaying
	musicChatChannelID := instance.Configuration.MusicTextChannelID
	songRequestID := instance.MusicData.CurrentSongRequest.ID
	songName := instance.MusicData.CurrentSong.SongName
	cancelSongFunc := instance.MusicData.CtxCancel
	instance.RUnlock()
	if !songPlaying {
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xff9999).
			AddField("Error skipping song.", "No song is currently playing.", false).
			SetThumbnail("https://img.icons8.com/arcade/64/playlist.png").
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send skip song error")
		return
	}
	_, err := instance.Db.Exec(`update song_request set played = true where id = $1`, songRequestID)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to update skipped song from the database.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xff9999).
			AddField("Error skipping song.", "Got an issue with the database.", false).
			SetThumbnail("https://img.icons8.com/arcade/64/playlist.png").
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send skip song error")
	}
	embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xffd9d9).
		AddField("Skipping current song...", songName, false).
		AddField("Requested By", message.Author.Username, false).
		SetThumbnail("https://img.icons8.com/arcade/64/playlist.png").
		MessageEmbed
	instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send song playing message.")
	cancelSongFunc()
}

func skipAllSongs(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	musicChatChannelID := instance.Configuration.MusicTextChannelID
	instance.RUnlock()

	skipAllCtx, skipAllCtxCancel := context.WithCancel(instance.Ctx)
	instance.MusicData.Lock()
	songRequestID := instance.MusicData.CurrentSongRequest.ID
	skipAllCtxCancelFunc := instance.MusicData.SkipAllCtxCancel
	skipAllCtxCancelFunc()
	instance.MusicData.SkipAllCtx = skipAllCtx
	instance.MusicData.SkipAllCtxCancel = skipAllCtxCancel
	instance.MusicData.Unlock()

	res, err := instance.Db.Exec(`update song_request set played = true where guild_id = $1 and played = false and id != $2`,
		instance.GuildID, songRequestID)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to delete skipped songs from the database.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xff9999).
			AddField("Error skipping all songs.", "Got an issue with the database.", false).
			SetThumbnail("https://img.icons8.com/arcade/64/playlist.png").
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send skip song error")
	}
	numDeleted, err := res.RowsAffected()
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to read rows effected.")
	}
	embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xffd9d9).
		AddField("Skipping all song requests.", fmt.Sprintf("Number skipped: %d", numDeleted), false).
		AddField("Requested By", message.Author.Username, false).
		SetThumbnail("https://img.icons8.com/arcade/64/playlist.png").
		MessageEmbed
	instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send song playing message.")

	skipSong(instance, message, args)
	instance.SendSongQueueEvent(music.SongQueueEvent{
		Song: nil, SongRequest: nil, Type: music.SongSkippedAll},
	)
}
