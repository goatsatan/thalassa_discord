package discord

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/sqlboiler/v4/boil"

	"thalassa_discord/models"
	"thalassa_discord/pkg/music"

	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func (serverInstance *ServerInstance) MuteUser(userID string) error {
	mutedRoleID, err := serverInstance.GetOrCreateMutedRole()
	if err != nil {
		return err
	}
	err = serverInstance.Session.GuildMemberRoleAdd(serverInstance.GuildID, userID, mutedRoleID)
	if err != nil {
		serverInstance.Log.Error().Str("muted_member", userID).
			Err(err).Msg("Unable to add muted role to user.")
		return err
	}

	return nil
}

func (serverInstance *ServerInstance) HandleSong(musicChatChannelID string) {
	serverInstance.MusicData.Lock()
	serverInstance.MusicData.SongPlaying = true
	serverInstance.MusicData.Unlock()

	defer func() {
		serverInstance.MusicData.Lock()
		serverInstance.MusicData.SongPlaying = false
		serverInstance.MusicData.Unlock()
	}()

songQueue:
	for {
		select {
		case <-serverInstance.Ctx.Done():
			return
		default:
			nextSongRequest, err := models.SongRequests(
				qm.Where("guild_id = ?", serverInstance.GuildID),
				qm.Where("played = false"),
				qm.OrderBy("requested_at ASC"),
				qm.Load(models.SongRequestRels.Song),
			).One(serverInstance.Ctx, serverInstance.Db)
			if err != nil {
				if err != sql.ErrNoRows {
					serverInstance.Log.Error().Err(err).Msg("Unable to query song requests.")
					return
				} else {
					break songQueue
				}
			}

			embedmsg := NewEmbedInfer(serverInstance.Session.State.User, 53503).
				AddField("Now Playing", fmt.Sprintf("[%s](%s)", nextSongRequest.R.Song.SongName, nextSongRequest.R.Song.URL), false).
				SetImage(nextSongRequest.R.Song.ThumbnailURL.String)

			song := nextSongRequest.R.Song
			if song.Artist.Valid {
				embedmsg.AddField("Artist", song.Artist.String, true)
			}
			if song.Album.Valid {
				embedmsg.AddField("Album", song.Album.String, true)
			}
			if song.Track.Valid {
				embedmsg.AddField("Track", song.Track.String, true)
			}
			if song.DurationInSeconds.Valid {
				minutes := song.DurationInSeconds.Int / 60
				seconds := song.DurationInSeconds.Int % 60
				secondPlur := "second"
				if seconds > 1 {
					secondPlur = "seconds"
				}
				minutePlur := "minute"
				if minutes > 1 {
					minutePlur = "minutes"
				}
				dur := ""
				if minutes > 0 {
					if seconds > 0 {
						dur = fmt.Sprintf("%d %s, %d %s", minutes, minutePlur, seconds, secondPlur)
					} else {
						dur = fmt.Sprintf("%d %s", minutes, minutePlur)
					}
				} else {
					dur = fmt.Sprintf("%d %s", seconds, secondPlur)
				}
				embedmsg.AddField("Duration", dur, false)
			}
			embedmsg.AddField("Requested By", nextSongRequest.UsernameAtTime, false)
			serverInstance.SendEmbedMessage(embedmsg.MessageEmbed, musicChatChannelID, "Unable to send song playing message.")

			serverInstance.Session.RLock()
			voiceConnection, exists := serverInstance.Session.VoiceConnections[serverInstance.GuildID]
			if !exists {
				serverInstance.Log.Error().Msg("Unable to find voice connection")
				serverInstance.Session.RUnlock()
				return
			}
			voiceReady := voiceConnection.Ready
			serverInstance.Session.RUnlock()
			if voiceReady {
				nextSongRequest.PlayedAt = null.TimeFrom(time.Now().UTC())
				ctx, ctxCancel := context.WithCancel(serverInstance.Ctx)
				serverInstance.MusicData.Lock()
				duration := 0
				if nextSongRequest.R.Song.DurationInSeconds.Valid {
					duration = nextSongRequest.R.Song.DurationInSeconds.Int
				}
				serverInstance.MusicData.SongDurationSeconds = duration
				serverInstance.MusicData.SongStarted = time.Now().UTC()
				serverInstance.MusicData.Ctx = ctx
				serverInstance.MusicData.CtxCancel = ctxCancel
				serverInstance.MusicData.CurrentSongRequestID = nextSongRequest.ID
				serverInstance.MusicData.CurrentSongName = nextSongRequest.SongName
				serverInstance.MusicData.Unlock()
				serverInstance.Log.Info().Msgf("Playing song: %s", nextSongRequest.SongName)
				music.StreamSong(ctx, nextSongRequest.R.Song.URL, serverInstance.Log, voiceConnection, serverInstance.Configuration.MusicVolume)

				// Don't mark the song as played if the bot is shutting down.
				select {
				case <-serverInstance.Ctx.Done():
					return
				default:
					nextSongRequest.Played = true
					_, err = nextSongRequest.Update(serverInstance.Ctx, serverInstance.Db, boil.Infer())
					if err != nil && !errors.Is(err, context.Canceled) {
						serverInstance.Log.Error().Err(err).Msg("Unable to update song")
						return
					}
				}
			} else {
				// TODO handle voice not ready.
				serverInstance.Log.Error().Msg("Voice not ready.")
				return
			}
		}
	}
}

// TODO refactor to handle re-joining voice when voice isn't ready.

func (serverInstance *ServerInstance) JoinVoice() error {
	serverInstance.RLock()
	defer serverInstance.RUnlock()
	if serverInstance.Configuration.MusicEnabled {
		_, err := serverInstance.Session.ChannelVoiceJoin(serverInstance.GuildID,
			serverInstance.Configuration.MusicVoiceChannelID.String, false, true)
		if err != nil {
			serverInstance.Log.Error().Err(err).Msg("Unable to join voice")
			return err
		} else {
			// If there's a song playing currently don't start playing another song.
			serverInstance.MusicData.RLock()
			songPlaying := serverInstance.MusicData.SongPlaying
			serverInstance.MusicData.RUnlock()
			if !songPlaying {
				serverInstance.HandleSong(serverInstance.Configuration.MusicTextChannelID.String)
			}
		}
	}
	return nil
}
