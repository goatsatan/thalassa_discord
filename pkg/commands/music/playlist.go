package music

import (
	"strconv"

	"thalassa_discord/pkg/discord"
	"thalassa_discord/pkg/music"

	"github.com/bwmarrin/discordgo"
)

func playList(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	musicChatChannelID := instance.Configuration.MusicTextChannelID
	instance.RUnlock()

	instance.MusicData.RLock()
	ctx := instance.MusicData.SkipAllCtx
	instance.MusicData.RUnlock()

	err := instance.Session.ChannelMessageDelete(message.ChannelID, message.ID)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to delete message.")
	}
	if len(args) == 0 {
		_, err := instance.Session.ChannelMessageSend(message.ChannelID, "You must specify a URL to a playlist.")
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to send channel message.")
		}
		return
	}

	embedmsg := discord.NewEmbedInfer(instance.Session.State.User, discord.DARKER_GREY).
		SetTitle("Attempting to parse a playlist").
		SetDescription("This can take a few minutes depending on the size of the playlist.").
		AddField("Requested by", message.Author.Username, false).
		SetThumbnail("https://img.icons8.com/arcade/64/playlist.png").
		MessageEmbed
	instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send message about song queue addition.")

	shufflePlaylist := false
	if len(args) > 1 && (args[1] == "shuffle" || args[1] == "random") {
		shufflePlaylist = true
	}

	playlistSongs, err := music.GetPlaylistInfo(instance.Ctx, args[0], shufflePlaylist)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get playlist info.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 0xff9999).
			AddField("Error getting playlist information:", err.Error(), false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send playlist info error message.")
		return
	}

	done := make(chan struct{})
	sendEmbedMessage := true
	if len(playlistSongs) > 25 {
		sendEmbedMessage = false
	}
	go func() {
		for index, songInfo := range playlistSongs {
			select {
			case <-ctx.Done():
				return
			default:
				handleSongInfo(instance, message, musicChatChannelID.String, songInfo, sendEmbedMessage)
				if index == 0 {
					close(done)
				}
			}
		}
	}()
	<-done
	// Send a single embed message when queueing more than 25 songs at a time.
	if !sendEmbedMessage {
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 28804).
			AddField("Number of playlist songs added to the queue", strconv.Itoa(len(playlistSongs)), false).
			AddField("Requested By", message.Author.Username, false).
			SetThumbnail("https://img.icons8.com/arcade/64/playlist.png").
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send song added to queue message.")
	}
	instance.MusicData.RLock()
	currentlyPlaying := instance.MusicData.SongPlaying
	instance.MusicData.RUnlock()
	if !currentlyPlaying {
		instance.TriggerNextSong <- struct{}{}
	}
}
