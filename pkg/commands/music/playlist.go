package music

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

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
	playlistSongs, err := music.GetPlaylistInfo(instance.Ctx, args[0])
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get playlist info.")
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Error getting playlist information:", err.Error(), false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, musicChatChannelID.String, "Unable to send playlist info error message.")
		return
	}
	workerLimit := make(chan struct{}, 2)
	wg := new(sync.WaitGroup)
	for index, song := range playlistSongs {
		select {
		case <-ctx.Done():
			// Wait for wait group to finish so we know we're skipping all songs.
			wg.Wait()
			// TODO break this code out into something re-usable.
			// Skip all
			instance.MusicData.RLock()
			songRequestID := instance.MusicData.CurrentSongRequestID
			instance.MusicData.RUnlock()
			_, err := instance.Db.Exec(`delete from song_request where guild_id = $1 and played = false and id != $2`,
				instance.GuildID, songRequestID)
			if err != nil {
				instance.Log.Error().Err(err).Msg("Unable to delete skipped songs from the database.")
			}
			return
		default:
			workerLimit <- struct{}{}
			wg.Add(1)
			go func(i *discord.ServerInstance, m *discordgo.Message, cID, url string, ix int) {
				handleSongInfo(i, m, cID, url)
				randSleepTime := 1 + rand.Intn(10-1)
				select {
				case <-ctx.Done():
				default:
					if ix > 2 {
						time.Sleep(time.Second * time.Duration(randSleepTime))
					}
				}
				<-workerLimit
				wg.Done()
			}(instance, message, musicChatChannelID.String, song.URL, index)

			// First 2 songs are queued up. Let's try to start playing.
			if index == 2 {
				go func() {
					instance.MusicData.RLock()
					currentlyPlaying := instance.MusicData.SongPlaying
					instance.MusicData.RUnlock()
					if !currentlyPlaying {
						instance.HandleSong(musicChatChannelID.String)
					}
				}()
			}
		}
	}
}
