package music

import (
	"fmt"
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
	_, msgErr := instance.Session.ChannelMessageSend(message.ChannelID,
		fmt.Sprintf("🎵 Attempting to parse a playlist that was requested by %s. 🎵\n"+
			" This can take a few minutes depending on the size of the playlist.",
			message.Author.Username))
	if msgErr != nil {
		instance.Log.Error().Err(msgErr).Msg("Unable to send message about playlist parsing.")
	}
	shufflePlaylist := false
	if len(args) > 1 && (args[1] == "shuffle" || args[1] == "random") {
		shufflePlaylist = true
	}

	playlistSongs, err := music.GetPlaylistInfo(instance.Ctx, args[0], shufflePlaylist)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get playlist info.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Error getting playlist information:", err.Error(), false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send playlist info error message.")
		return
	}

	done := make(chan struct{})
	go func() {
		for index, songInfo := range playlistSongs {
			select {
			case <-ctx.Done():
				return
			default:
				handleSongInfo(instance, message, musicChatChannelID.String, songInfo)
				if index == 0 {
					close(done)
				}
			}
		}
	}()
	<-done
	instance.MusicData.RLock()
	currentlyPlaying := instance.MusicData.SongPlaying
	instance.MusicData.RUnlock()
	if !currentlyPlaying {
		instance.HandleSong(musicChatChannelID.String)
	}
}
