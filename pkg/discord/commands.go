package discord

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
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

const (
	AQUA                = 1752220
	GREEN               = 3066993
	BLUE                = 3447003
	PURPLE              = 10181046
	GOLD                = 15844367
	ORANGE              = 15105570
	RED                 = 15158332
	GREY                = 9807270
	DARKER_GREY         = 8359053
	NAVY                = 3426654
	DARK_AQUA           = 1146986
	DARK_GREEN          = 2067276
	DARK_BLUE           = 2123412
	DARK_PURPLE         = 7419530
	DARK_GOLD           = 12745742
	DARK_ORANGE         = 11027200
	DARK_RED            = 10038562
	DARK_GREY           = 9936031
	LIGHT_GREY          = 12370112
	DARK_NAVY           = 2899536
	LUMINOUS_VIVID_PINK = 16580705
	DARK_VIVID_PINK     = 12320855
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
	s.registerCommand(
		Command{
			Name:     "dog",
			HelpText: "This gets a random picture of a dog.",
			Execute:  getRandomDogPicture,
		})
	s.registerCommand(
		Command{
			Name:     "cat",
			HelpText: "This gets a random picture of a cat.",
			Execute:  getRandomCatPicture,
		})
	s.registerCommand(
		Command{
			Name:     "fox",
			HelpText: "This gets a random picture of a fox.",
			Execute:  getRandomFoxPicture,
		})
	s.registerCommand(
		Command{
			Name:     "define",
			HelpText: "This gets the definition for a word.",
			Execute:  getDictionary,
		})
	s.registerCommand(
		Command{
			Name:     "joke",
			HelpText: "This gets a random joke. If you're easily offended use safejoke instead.",
			Execute:  getRandomJoke,
		})
	s.registerCommand(
		Command{
			Name:     "safejoke",
			HelpText: "This gets a random joke.",
			Execute:  getRandomSafeJoke,
		})
	s.registerCommand(
		Command{
			Name:     "iplookup",
			HelpText: "This gets IP address information",
			Execute:  ipLookup,
		})
	s.registerCommand(
		Command{
			Name:     "udefine",
			HelpText: "Urban dictionary lookup.",
			Execute:  urbanDictionary,
		})
}

func (s *shardInstance) registerCommand(command Command) {
	s.Lock()
	s.Commands[strings.ToLower(command.Name)] = &command
	s.Unlock()
}

func sendErrorEmbed(instance *ServerInstance, errorMessage, errorDescription, channelID string) {
	embedmsg := NewEmbedInfer(instance.Session.State.User.Username, RED).
		AddField(errorMessage, errorDescription, false).
		MessageEmbed
	SendEmbedMessage(instance, embedmsg, channelID, errorMessage)
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

func getRandomDogPicture(instance *ServerInstance, message *discordgo.Message, args []string) {
	type dogJSONResponse struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}
	resp, err := instance.httpClient.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get random dog image.")
		sendErrorEmbed(instance, "Unable to get random dog image.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := dogJSONResponse{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to parse JSON from dog API.")
		sendErrorEmbed(instance, "Unable to parse JSON from dog API.", err.Error(), message.ChannelID)
		return
	}

	dogImage := respJSON.Message
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, dogImage)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send dog message.")
	}
}

func getRandomFoxPicture(instance *ServerInstance, message *discordgo.Message, args []string) {
	type foxJSONResponse struct {
		Image string `json:"image"`
		Link  string `json:"link"`
	}

	resp, err := instance.httpClient.Get("https://randomfox.ca/floof/")
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get random fox image.")
		sendErrorEmbed(instance, "Unable to get random fox image.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := foxJSONResponse{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to parse JSON from fox API.")
		sendErrorEmbed(instance, "Unable to parse JSON from fox API.", err.Error(), message.ChannelID)
		return
	}

	dogImage := respJSON.Image
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, dogImage)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send fox message.")
	}
}

func getRandomCatPicture(instance *ServerInstance, message *discordgo.Message, args []string) {
	type catJSONResponse struct {
		File string `json:"file"`
	}

	resp, err := instance.httpClient.Get("https://aws.random.cat/meow")
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get random cat image.")
		sendErrorEmbed(instance, "Unable to get random cat image.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := catJSONResponse{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to parse JSON from cat API.")
		sendErrorEmbed(instance, "Unable to parse JSON from cat API.", err.Error(), message.ChannelID)
		return
	}

	dogImage := respJSON.File
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, dogImage)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send fox message.")
	}
}

func getDictionary(instance *ServerInstance, message *discordgo.Message, args []string) {
	type dictionaryResponse struct {
		Definitions []struct {
			Type       string      `json:"type"`
			Definition string      `json:"definition"`
			Example    *string     `json:"example"`
			ImageURL   string      `json:"image_url"`
			Emoji      interface{} `json:"emoji"`
		} `json:"definitions"`
		Word          string `json:"word"`
		Pronunciation string `json:"pronunciation"`
	}

	if len(args) == 0 {
		sendErrorEmbed(instance, "You must provide a word to lookup", "No word provided",
			message.ChannelID)
		return
	}

	url := fmt.Sprintf("https://owlbot.info/api/v4/dictionary/%s", args[0])

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to create GET request for dictionary.")
		sendErrorEmbed(instance, "Unable to lookup word", err.Error(),
			message.ChannelID)
		return
	}
	req.Header.Add("Authorization", "Token 2caaf1f54e8c0d10f7a345e6af45aa8c7beeeb50")
	resp, err := instance.httpClient.Do(req)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to lookup word.")
		sendErrorEmbed(instance, "Unable to lookup word.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to read response body of dictionary lookup.")
		sendErrorEmbed(instance, "Unable to parse JSON from dictionary API.", err.Error(), message.ChannelID)
		return
	}
	if string(bodyBytes) == `[{"message":"No definition :("}]` {
		sendErrorEmbed(instance, "No definition was found", fmt.Sprintf("Definition for %s was not found", args[0]), message.ChannelID)
		return
	}
	jsonDecoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	respJSON := dictionaryResponse{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).WithField("Body", string(bodyBytes)).Error("Unable to parse JSON from dictionary API.")
		sendErrorEmbed(instance, "Unable to parse JSON from dictionary API.", err.Error(), message.ChannelID)
		return
	}

	if len(respJSON.Definitions) == 0 {
		sendErrorEmbed(instance, "No definitions found", "No definitions found", message.ChannelID)
		return
	}

	embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 28804).
		AddField("Word", args[0], false).
		AddField("Description", respJSON.Definitions[0].Definition, false).
		SetImage(respJSON.Definitions[0].ImageURL).
		MessageEmbed
	SendEmbedMessage(instance, embedmsg, message.ChannelID, "Unable to send dictionary message.")
}

func getRandomJoke(instance *ServerInstance, message *discordgo.Message, args []string) {
	type jokeJSON struct {
		Error    bool   `json:"error"`
		Category string `json:"category"`
		Type     string `json:"type"`
		Setup    string `json:"setup"`
		Delivery string `json:"delivery"`
		Flags    struct {
			Nsfw      bool `json:"nsfw"`
			Religious bool `json:"religious"`
			Political bool `json:"political"`
			Racist    bool `json:"racist"`
			Sexist    bool `json:"sexist"`
		} `json:"flags"`
		ID   int    `json:"id"`
		Lang string `json:"lang"`
	}

	resp, err := instance.httpClient.Get("https://sv443.net/jokeapi/v2/joke/Any?blacklistFlags=racist&type=twopart")
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get random joke.")
		sendErrorEmbed(instance, "Unable to get random joke.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := jokeJSON{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to parse JSON from joke API.")
		sendErrorEmbed(instance, "Unable to parse JSON from joke API.", err.Error(), message.ChannelID)
		return
	}

	jokeString := fmt.Sprintf("%s \n %s", respJSON.Setup, respJSON.Delivery)
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, jokeString)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send joke message.")
	}
}

func getRandomSafeJoke(instance *ServerInstance, message *discordgo.Message, args []string) {
	type jokeJSON struct {
		Error    bool   `json:"error"`
		Category string `json:"category"`
		Type     string `json:"type"`
		Setup    string `json:"setup"`
		Delivery string `json:"delivery"`
		Flags    struct {
			Nsfw      bool `json:"nsfw"`
			Religious bool `json:"religious"`
			Political bool `json:"political"`
			Racist    bool `json:"racist"`
			Sexist    bool `json:"sexist"`
		} `json:"flags"`
		ID   int    `json:"id"`
		Lang string `json:"lang"`
	}

	resp, err := instance.httpClient.Get("https://sv443.net/jokeapi/v2/joke/Any?blacklistFlags=nsfw,religious,political,racist,sexist&type=twopart")
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get random joke.")
		sendErrorEmbed(instance, "Unable to get random joke.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := jokeJSON{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to parse JSON from joke API.")
		sendErrorEmbed(instance, "Unable to parse JSON from joke API.", err.Error(), message.ChannelID)
		return
	}

	jokeString := fmt.Sprintf("%s \n %s", respJSON.Setup, respJSON.Delivery)
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, jokeString)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send joke message.")
	}
}

func ipLookup(instance *ServerInstance, message *discordgo.Message, args []string) {
	type ipJSON struct {
		IP                 string  `json:"ip"`
		City               string  `json:"city"`
		Region             string  `json:"region"`
		RegionCode         string  `json:"region_code"`
		Country            string  `json:"country"`
		CountryCode        string  `json:"country_code"`
		CountryCodeIso3    string  `json:"country_code_iso3"`
		CountryCapital     string  `json:"country_capital"`
		CountryTld         string  `json:"country_tld"`
		CountryName        string  `json:"country_name"`
		ContinentCode      string  `json:"continent_code"`
		InEu               bool    `json:"in_eu"`
		Postal             string  `json:"postal"`
		Latitude           float64 `json:"latitude"`
		Longitude          float64 `json:"longitude"`
		Timezone           string  `json:"timezone"`
		UtcOffset          string  `json:"utc_offset"`
		CountryCallingCode string  `json:"country_calling_code"`
		Currency           string  `json:"currency"`
		CurrencyName       string  `json:"currency_name"`
		Languages          string  `json:"languages"`
		CountryArea        float64 `json:"country_area"`
		CountryPopulation  float64 `json:"country_population"`
		Asn                string  `json:"asn"`
		Org                string  `json:"org"`
	}

	type errorJSON struct {
		IP     string `json:"ip"`
		Error  bool   `json:"error"`
		Reason string `json:"reason"`
	}

	if len(args) == 0 {
		sendErrorEmbed(instance, "Invalid IP/Hostname.", "You must provide an address.", message.ChannelID)
		return
	}

	url := fmt.Sprintf("https://ipapi.co/%s/json/", args[0])

	resp, err := instance.httpClient.Get(url)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get IP information.")
		sendErrorEmbed(instance, "Unable to get IP information.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to read response body of IP lookup.")
		sendErrorEmbed(instance, "Unable to parse JSON from IP API.", err.Error(), message.ChannelID)
		return
	}
	jsonDecoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	respJSON := ipJSON{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		errJSON := errorJSON{}
		errDecoder := json.NewDecoder(bytes.NewReader(bodyBytes))
		err = errDecoder.Decode(&errJSON)
		if err != nil {
			instance.Log.WithError(err).Error("Unable to parse JSON from IP lookup.")
			sendErrorEmbed(instance, "Unable to parse JSON from IP lookup.", err.Error(), message.ChannelID)
			return
		}
		sendErrorEmbed(instance, "Unable to lookup IP information", errJSON.Reason, message.ChannelID)
	}

	embedmsg := NewEmbedInfer(instance.Session.State.User.Username, 28804).
		AddField("IP", respJSON.IP, false).
		AddField("City", respJSON.City, true).
		AddField("Region", respJSON.Region, true).
		AddField("Country", respJSON.Country, true).
		AddField("ASN", respJSON.Asn, false).
		AddField("ISP", respJSON.Org, true).
		MessageEmbed
	SendEmbedMessage(instance, embedmsg, message.ChannelID, "Unable to send ip lookup message.")
}

func urbanDictionary(instance *ServerInstance, message *discordgo.Message, args []string) {
	type dictionaryJSON struct {
		List []struct {
			Definition  string    `json:"definition"`
			Permalink   string    `json:"permalink"`
			ThumbsUp    int       `json:"thumbs_up"`
			SoundUrls   []string  `json:"sound_urls"`
			Author      string    `json:"author"`
			Word        string    `json:"word"`
			Defid       int       `json:"defid"`
			CurrentVote string    `json:"current_vote"`
			WrittenOn   time.Time `json:"written_on"`
			Example     string    `json:"example"`
			ThumbsDown  int       `json:"thumbs_down"`
		} `json:"list"`
	}

	if len(args) == 0 {
		sendErrorEmbed(instance, "Invalid word", "You must provide a word.", message.ChannelID)
		return
	}

	url := fmt.Sprintf("https://mashape-community-urban-dictionary.p.rapidapi.com/define?term=%s", args[0])

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to create GET request for dictionary.")
		sendErrorEmbed(instance, "Unable to lookup word", err.Error(),
			message.ChannelID)
		return
	}
	req.Header.Add("x-rapidapi-host", "mashape-community-urban-dictionary.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", "cedd53e93amsh465918faf691d51p16006fjsnbf8ea937fa69")
	resp, err := instance.httpClient.Do(req)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to lookup word.")
		sendErrorEmbed(instance, "Unable to lookup word.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to read response body of dictionary lookup.")
		sendErrorEmbed(instance, "Unable to parse JSON from dictionary API.", err.Error(), message.ChannelID)
		return
	}
	jsonDecoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	respJSON := dictionaryJSON{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to parse JSON from dictionary lookup.")
		sendErrorEmbed(instance, "Unable to parse JSON from dictionary lookup.", err.Error(), message.ChannelID)
		return
	}

	for i, definition := range respJSON.List {
		if i >= 3 {
			break
		}
		embedmsg := NewEmbedInfer(instance.Session.State.User.Username, AQUA).
			AddField("Definition", definition.Definition, false).
			AddField("Example", definition.Example, false).
			MessageEmbed
		embedmsg.Title = fmt.Sprintf("%s definition %d", definition.Word, i+1)
		embedmsg.URL = definition.Permalink
		SendEmbedMessage(instance, embedmsg, message.ChannelID, "Unable to send urban dictionary lookup message.")
	}
}
