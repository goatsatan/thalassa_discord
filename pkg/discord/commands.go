package discord

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"thalassa_discord/models"
	"thalassa_discord/pkg/music"

	"github.com/bwmarrin/discordgo"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

type Command struct {
	Name     string
	HelpText string
	Execute  func(*ServerInstance, *discordgo.Message, []string)
}

func (s *shardInstance) registerBuiltInCommands() {
	s.registerCommand(
		Command{
			Name:     "play",
			HelpText: "This command takes a URL and tries to play the audio in voice chat.",
			Execute:  playSong,
		})
	s.registerCommand(
		Command{
			Name:     "skip",
			HelpText: "This skips the current playing song.",
			Execute:  skipSong,
		})
	s.registerCommand(
		Command{
			Name:     "skipall",
			HelpText: "This skips all songs in the queue as well as the current playing song.",
			Execute:  skipAllSongs,
		})
	s.registerCommand(
		Command{
			Name:     "playlist",
			HelpText: "This skips all songs in the queue as well as the current playing song.",
			Execute:  playList,
		})
}

func (s *shardInstance) registerCommand(command Command) {
	s.Lock()
	s.Commands[strings.ToLower(command.Name)] = &command
	s.Unlock()
}

func (s *shardInstance) parseMessageForCommand(message *discordgo.Message, instance *ServerInstance) {
	if len(message.Content) > 0 {
		instance.RLock()
		prefix := instance.Configuration.PrefixCommand
		instance.RUnlock()
		if string(message.Content[0]) == prefix {
			splitMsg := strings.Split(message.Content, " ")
			commandName := splitMsg[0]
			commandName = strings.ToLower(commandName)
			commandName = commandName[1:]
			args := splitMsg[1:]
			command, exists := s.Commands[commandName]
			if exists {
				command.Execute(instance, message, args)
			}
		}
	}
}

func interfaceToNullString(x interface{}) null.String {
	stringParse, ok := x.(string)
	if !ok {
		return null.String{}
	}
	return null.StringFrom(stringParse)
}

func skipSong(instance *ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	musicChatChannelID := instance.Configuration.MusicTextChannelID
	instance.RUnlock()
	instance.MusicData.RLock()
	songRequestID := instance.MusicData.CurrentSongRequestID
	songName := instance.MusicData.CurrentSongName
	cancelSongFunc := instance.MusicData.CtxCancel
	instance.MusicData.RUnlock()
	_, err := instance.db.Exec(`update song_request set played = true where id = $1`, songRequestID)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to update skipped song from the database.")
		embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Error skipping song.", "Got an issue with the database.", false).
			MessageEmbed
		SendEmbedMessage(instance, embedmsg, musicChatChannelID.String, "Unable to send skip song error")
	}
	embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 0xffd9d9).
		AddField("Skipping current song...", songName, false).
		AddField("Requested By", message.Author.Username, false).
		MessageEmbed
	SendEmbedMessage(instance, embedmsg, musicChatChannelID.String, "Unable to send song playing message.")
	cancelSongFunc()
}

func skipAllSongs(instance *ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	musicChatChannelID := instance.Configuration.MusicTextChannelID
	instance.RUnlock()

	skipAllCtx, skipAllCtxCancel := context.WithCancel(context.Background())
	instance.MusicData.Lock()
	songRequestID := instance.MusicData.CurrentSongRequestID
	skipAllCtxCancelFunc := instance.MusicData.SkipAllCtxCancel
	skipAllCtxCancelFunc()
	instance.MusicData.SkipAllCtx = skipAllCtx
	instance.MusicData.SkipAllCtxCancel = skipAllCtxCancel
	instance.MusicData.Unlock()

	res, err := instance.db.Exec(`delete from song_request where guild_id = $1 and played = false and id != $2`,
		instance.Guild.ID, songRequestID)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to delete skipped songs from the database.")
		embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Error skipping all songs.", "Got an issue with the database.", false).
			MessageEmbed
		SendEmbedMessage(instance, embedmsg, musicChatChannelID.String, "Unable to send skip song error")
	}
	numDeleted, err := res.RowsAffected()
	if err != nil {
		instance.Log.WithError(err).Error("Unable to read rows effected.")
	}
	embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 0xffd9d9).
		AddField("Skipping all song requests.", fmt.Sprintf("Number skipped: %d", numDeleted), false).
		AddField("Requested By", message.Author.Username, false).
		MessageEmbed
	SendEmbedMessage(instance, embedmsg, musicChatChannelID.String, "Unable to send song playing message.")

	skipSong(instance, message, args)
}

func playList(instance *ServerInstance, message *discordgo.Message, args []string) {
	instance.RLock()
	musicChatChannelID := instance.Configuration.MusicTextChannelID
	instance.RUnlock()

	instance.MusicData.RLock()
	ctx := instance.MusicData.SkipAllCtx
	instance.MusicData.RUnlock()

	err := instance.Session.ChannelMessageDelete(message.ChannelID, message.ID)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to delete message.")
	}
	if len(args) == 0 {
		_, err := instance.Session.ChannelMessageSend(message.ChannelID, "You must specify a URL to a playlist.")
		if err != nil {
			instance.Log.WithError(err).Error("Unable to send channel message.")
		}
		return
	}
	_, msgErr := instance.Session.ChannelMessageSend(message.ChannelID,
		fmt.Sprintf("🎵 Attempting to parse a playlist that was requested by %s. 🎵\n"+
			" This can take a few minutes depending on the size of the playlist.",
			message.Author.Username))
	if msgErr != nil {
		instance.Log.WithError(msgErr).Error("Unable to send message about playlist parsing.")
	}
	playlistSongs, err := music.GetPlaylistInfo(context.Background(), args[0])
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get playlist info.")
		embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Error getting playlist information:", err.Error(), false).
			MessageEmbed
		SendEmbedMessage(instance, embedmsg, musicChatChannelID.String, "Unable to send playlist info error message.")
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
			_, err := instance.db.Exec(`delete from song_request where guild_id = $1 and played = false and id != $2`,
				instance.Guild.ID, songRequestID)
			if err != nil {
				instance.Log.WithError(err).Error("Unable to delete skipped songs from the database.")
			}
			return
		default:
			workerLimit <- struct{}{}
			wg.Add(1)
			go func(i *ServerInstance, m *discordgo.Message, cID, url string, ix int) {
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
						handleSong(instance, musicChatChannelID.String)
					}
				}()
			}
		}
	}
}

func playSong(instance *ServerInstance, message *discordgo.Message, args []string) {
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
		handleSong(instance, musicChatChannelID.String)
	}
}

func handleSongInfo(instance *ServerInstance, message *discordgo.Message, musicChatChannelID, url string) {
	// fmt.Println("Handling song info.")
	song, err := music.GetSongInfo(context.Background(), url)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get song info.")
		embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Error getting song information:", err.Error(), false).
			MessageEmbed
		SendEmbedMessage(instance, embedmsg, musicChatChannelID, "Unable to send song info error message.")
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
		Artist:            interfaceToNullString(song.Artist),
		Album:             interfaceToNullString(song.Album),
		Track:             interfaceToNullString(song.Track),
	}
	err = newSong.Upsert(context.Background(), instance.db, true, []string{"id"}, boil.Infer(), boil.Infer())
	if err != nil {
		instance.Log.WithError(err).Error("Unable to upsert song.")
		embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Unable to add song request.", "Database error.", false).
			MessageEmbed
		SendEmbedMessage(instance, embedmsg, musicChatChannelID, "Unable to upsert song.")
		return
	}
	newSongRequest := &models.SongRequest{
		SongID:            null.StringFrom(newSong.ID),
		SongName:          song.Title,
		RequestedByUserID: message.Author.ID,
		UsernameAtTime:    message.Author.Username,
		GuildID:           message.GuildID,
		GuildNameAtTime:   instance.Guild.Name,
		RequestedAt:       time.Now().UTC(),
		PlayedAt:          null.Time{},
		Played:            false,
	}
	err = newSongRequest.Insert(context.Background(), instance.db, boil.Infer())
	if err != nil {
		instance.Log.WithError(err).Error("Unable to insert song request.")
		embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 0xff9999).
			AddField("Unable to add song request.", "Database error.", false).
			MessageEmbed
		SendEmbedMessage(instance, embedmsg, musicChatChannelID, "Unable to insert song request.")
		return
	}
	embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 28804).
		AddField("Song has been added to the queue", song.Title, false).
		AddField("Requested By", message.Author.Username, false).
		MessageEmbed
	SendEmbedMessage(instance, embedmsg, musicChatChannelID, "Unable to send song added to queue message.")
	return
}

func handleSong(instance *ServerInstance, musicChatChannelID string) {
	instance.MusicData.Lock()
	instance.MusicData.SongPlaying = true
	instance.MusicData.Unlock()

	defer func() {
		instance.MusicData.Lock()
		instance.MusicData.SongPlaying = false
		instance.MusicData.Unlock()
	}()

songQueue:
	for {
		select {
		case <-instance.Ctx.Done():
			return
		default:
			nextSongRequest, err := models.SongRequests(
				qm.Where("guild_id = ?", instance.Guild.ID),
				qm.Where("played = false"),
				qm.OrderBy("requested_at ASC"),
				qm.Load(models.SongRequestRels.Song),
			).One(context.Background(), instance.db)
			if err != nil {
				if err != sql.ErrNoRows {
					instance.Log.WithError(err).Error("Unable to query song requests.")
					return
				} else {
					break songQueue
				}
			}
			instance.Log.Infof("Playing song: %s", nextSongRequest.SongName)

			embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 53503).
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
			SendEmbedMessage(instance, embedmsg.MessageEmbed, musicChatChannelID, "Unable to send song playing message.")

			instance.Session.RLock()
			voiceConnection, exists := instance.Session.VoiceConnections[instance.Guild.ID]
			if !exists {
				instance.Log.Error("Unable to find voice connection")
				instance.Session.RUnlock()
				return
			}
			voiceReady := voiceConnection.Ready
			instance.Session.RUnlock()
			if voiceReady {
				nextSongRequest.PlayedAt = null.TimeFrom(time.Now().UTC())
				ctx, ctxCancel := context.WithCancel(context.Background())
				instance.MusicData.Lock()
				duration := 0
				if nextSongRequest.R.Song.DurationInSeconds.Valid {
					duration = nextSongRequest.R.Song.DurationInSeconds.Int
				}
				instance.MusicData.SongDurationSeconds = duration
				instance.MusicData.SongStarted = time.Now().UTC()
				instance.MusicData.Ctx = ctx
				instance.MusicData.CtxCancel = ctxCancel
				instance.MusicData.CurrentSongRequestID = nextSongRequest.ID
				instance.MusicData.CurrentSongName = nextSongRequest.SongName
				instance.MusicData.Unlock()
				music.StreamSong(ctx, nextSongRequest.R.Song.URL, instance.Log, voiceConnection, instance.Configuration.MusicVolume)
				nextSongRequest.Played = true
				_, err = nextSongRequest.Update(context.Background(), instance.db, boil.Infer())
				if err != nil {
					instance.Log.WithError(err).Error("Unable to update song")
					return
				}
			} else {
				// TODO handle voice not ready.
				instance.Log.Error("Voice not ready.")
				return
			}
		}
	}
}
