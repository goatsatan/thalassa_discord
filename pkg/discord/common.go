package discord

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/avast/retry-go"
	"github.com/friendsofgo/errors"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"thalassa_discord/models"
	"thalassa_discord/pkg/music"
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

func (serverInstance *ServerInstance) getNextSongInQueue() (*models.SongRequest, error) {
	nextSongRequest, err := models.SongRequests(
		qm.Where("guild_id = ?", serverInstance.GuildID),
		qm.Where("played = false"),
		qm.OrderBy("requested_at ASC"),
		qm.OrderBy("id asc"),
		qm.Load(models.SongRequestRels.Song),
	).One(serverInstance.Ctx, serverInstance.Db)
	if err != nil {
		return nil, err
	}
	return nextSongRequest, nil
}

func (serverInstance *ServerInstance) SendSongQueueEvent(songRequestEvent music.SongQueueEvent) {
	serverInstance.SongQueueUpdateCallbackMutex.RLock()
	defer serverInstance.SongQueueUpdateCallbackMutex.RUnlock()
	if serverInstance.SongQueueUpdateCallback != nil {
		serverInstance.SongQueueUpdateCallback(serverInstance.GuildID, songRequestEvent)
	}
}

func (serverInstance *ServerInstance) loopNextSongs(ctx context.Context, musicTextChannelID string) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			nextSongRequest, err := serverInstance.getNextSongInQueue()
			if err != nil || nextSongRequest == nil {
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					serverInstance.Log.Error().Err(err).Msg("Unable to get next song in queue")
				}
				return nil
			}
			serverInstance.Log.Debug().Str("song", nextSongRequest.R.Song.SongName).Msg("Playing next song")
			errPlaySong := serverInstance.handleSongRequest(musicTextChannelID, nextSongRequest)
			if errPlaySong != nil {
				serverInstance.Log.Error().Err(errPlaySong).Msg("Unable to play next song")
				return err
			}
		}
	}
}

func drainSongRequestTriggers(serverInstance *ServerInstance) {
Drain:
	for {
		select {
		case _, ok := <-serverInstance.TriggerNextSong:
			if !ok {
				break Drain
			}
		default:
			break Drain
		}
	}
}

func (serverInstance *ServerInstance) handleSongQueue() error {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		serverInstance.RLock()
		musicTextChannelID := serverInstance.Configuration.MusicTextChannelID.String
		serverInstance.RUnlock()
		select {
		case <-serverInstance.Ctx.Done():
			return nil
		case <-ticker.C:
			_ = serverInstance.loopNextSongs(serverInstance.Ctx, musicTextChannelID)
		case <-serverInstance.TriggerNextSong:
			// Drain other song request triggers, since we loop through all requests.
			drainSongRequestTriggers(serverInstance)
			_ = serverInstance.loopNextSongs(serverInstance.Ctx, musicTextChannelID)
		}
	}
}

func (serverInstance *ServerInstance) handleSongRequest(musicChatChannelID string, songRequest *models.SongRequest) error {
	embedmsg := NewEmbedInfer(serverInstance.Session.State.User, 53503).
		AddField("Now Playing", fmt.Sprintf("[%s](%s)", songRequest.R.Song.SongName, songRequest.R.Song.URL), false).
		SetImage(songRequest.R.Song.ThumbnailURL.String)

	song := songRequest.R.Song
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
	embedmsg.AddField("Requested By", songRequest.UsernameAtTime, false)
	serverInstance.SendEmbedMessage(embedmsg.MessageEmbed, musicChatChannelID, "Unable to send song playing message.")

	serverInstance.Session.RLock()
	voiceConnection, exists := serverInstance.Session.VoiceConnections[serverInstance.GuildID]
	if !exists {
		serverInstance.Log.Error().Msg("Unable to find voice connection")
		serverInstance.Session.RUnlock()
		return errors.New("unable to find voice connection")
	}
	voiceReady := voiceConnection.Ready
	serverInstance.Session.RUnlock()
	if voiceReady {
		songRequest.PlayedAt = null.TimeFrom(time.Now().UTC())
		ctx, ctxCancel := context.WithCancel(serverInstance.Ctx)
		serverInstance.MusicData.Lock()
		duration := 0
		if songRequest.R.Song.DurationInSeconds.Valid {
			duration = songRequest.R.Song.DurationInSeconds.Int
		}
		serverInstance.MusicData.SongDurationSeconds = duration
		serverInstance.MusicData.SongStarted = time.Now().UTC()
		serverInstance.MusicData.SongPlaying = true
		serverInstance.MusicData.Ctx = ctx
		serverInstance.MusicData.CtxCancel = ctxCancel
		serverInstance.MusicData.CurrentSongRequest = songRequest
		serverInstance.MusicData.CurrentSong = songRequest.R.Song

		// Send the song playing event to the song queue channel.
		serverInstance.SendSongQueueEvent(music.SongQueueEvent{
			Song: songRequest.R.Song, SongRequest: songRequest, Type: music.SongPlaying},
		)

		serverInstance.MusicData.Unlock()

		serverInstance.Log.Info().Msgf("Playing song: %s", songRequest.SongName)
		music.StreamSong(ctx, songRequest.R.Song.URL, serverInstance.Log, voiceConnection, serverInstance.Configuration.MusicVolume)
		serverInstance.MusicData.Lock()
		serverInstance.MusicData.SongPlaying = false

		// Send the song finished event to the song queue channel.
		songQueueEvent := music.SongQueueEvent{Song: songRequest.R.Song, SongRequest: songRequest, Type: music.SongFinished}
		if ctx.Err() != nil {
			// If the context was cancelled, the song was skipped.
			songQueueEvent.Type = music.SongSkipped
		}
		serverInstance.SendSongQueueEvent(songQueueEvent)

		serverInstance.MusicData.Unlock()

		// Don't mark the song as played if the bot is shutting down.
		select {
		case <-serverInstance.Ctx.Done():
			return nil
		default:
			songRequest.Played = true
			_, errUpdate := songRequest.Update(serverInstance.Ctx, serverInstance.Db, boil.Infer())
			if errUpdate != nil && !errors.Is(errUpdate, context.Canceled) {
				serverInstance.Log.Error().Err(errUpdate).Msg("Unable to update song")
				return errUpdate
			}
		}
	} else {
		serverInstance.Log.Error().Msg("Voice not ready.")
		return errors.New("voice not ready")
	}
	return nil
}

func (serverInstance *ServerInstance) ConnectToVoice() error {
	retryOptions := []retry.Option{
		retry.Context(serverInstance.Ctx),
		retry.Attempts(10),
		retry.Delay(1 * time.Second),
		retry.MaxDelay(1 * time.Minute),
	}
	errRetry := retry.Do(func() error {
		serverInstance.Session.RLock()
		voiceConnection, exists := serverInstance.Session.VoiceConnections[serverInstance.GuildID]
		guildID := serverInstance.GuildID
		musicVoiceChannelID := serverInstance.Configuration.MusicVoiceChannelID.String
		serverInstance.Session.RUnlock()
		if !exists {
			vc, errConnectVoice := serverInstance.Session.ChannelVoiceJoin(guildID,
				musicVoiceChannelID, false, true)
			if errConnectVoice != nil {
				return errConnectVoice
			}
			voiceConnection = vc
		}
		serverInstance.Session.RLock()
		voiceReady := voiceConnection.Ready
		serverInstance.Session.RUnlock()
		if !voiceReady {
			return errors.New("voice not ready")
		}
		return nil
	}, retryOptions...)
	return errRetry
}
