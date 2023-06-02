package music

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wader/goutubedl"
	"io"
	"math"
	"os/exec"
	"time"
)

func StreamSong(ctx context.Context, link string, log zerolog.Logger, vc *discordgo.VoiceConnection, volume float32) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"
	options.Volume = int(math.Round(float64(volume) * 100))
	log.Debug().Msgf("Streaming song %s", link)

	song, errGetSong := goutubedl.New(ctx, link, goutubedl.Options{})
	if errGetSong != nil {
		log.Error().Err(errGetSong).Msg("error getting song")
		return
	}

	songReader, errReader := song.Download(ctx, "bestaudio*")
	if errReader != nil {
		log.Error().Err(errReader).Msg("error reading song")
		return
	}
	defer func() {
		errClose := songReader.Close()
		if errClose != nil {
			log.Error().Err(errClose).Msg("error closing song reader")
		}
	}()

	errSpeaking := vc.Speaking(true)
	if errSpeaking != nil {
		log.Error().Err(errSpeaking).Msg("error setting speaking to true")
		return
	}

	defer func() {
		vc.RLock()
		ready := vc.Ready
		vc.RUnlock()
		if !ready {
			return
		}
		errSpeaking = vc.Speaking(false)
		if errSpeaking != nil {
			log.Error().Err(errSpeaking).Msg("error setting speaking to false")
		}
	}()

	encoding, errEncode := dca.EncodeMem(songReader, options)
	if errEncode != nil {
		log.Error().Err(errEncode).Msg("error encoding song")
		return
	}
	defer encoding.Cleanup()
	streamChan := make(chan error)
	dca.NewStream(encoding, vc, streamChan)
	select {
	case <-ctx.Done():
		log.Debug().Msg("song was skipped")
		return
	case err := <-streamChan:
		vc.RLock()
		ready := vc.Ready
		vc.RUnlock()
		if err != nil && err != io.EOF && ready {
			log.Error().Err(err).Msg("error streaming song")
		}
	}
}

func GetSongInfo(ctx context.Context, url string) (*goutubedl.Info, error) {
	song, errInfo := goutubedl.New(ctx, url, goutubedl.Options{})
	if errInfo != nil {
		return nil, errInfo
	}
	return &song.Info, nil
}

func GetPlaylistInfo(ctx context.Context, url string) ([]*goutubedl.Info, error) {
	ytdlCtx, ytdlCtxCancel := context.WithTimeout(ctx, time.Minute*5)
	defer ytdlCtxCancel()
	ytdlpArgs := []string{
		"--dump-json",
		"--flat-playlist",
		"--no-progress",
		"--default-search",
		"ytsearch",
		"--no-call-home",
		"--skip-download",
		url,
	}
	cmd := exec.CommandContext(ytdlCtx, "yt-dlp", ytdlpArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Msg("error getting playlist info")
		return nil, err
	}
	var playListSongs []*goutubedl.Info
	songs := bytes.Split(output, []byte("\n"))
	for _, song := range songs {
		s := song
		if s == nil || len(s) == 0 {
			continue
		}
		songInfo := &goutubedl.Info{}
		errUnmarshal := json.Unmarshal(s, songInfo)
		if errUnmarshal != nil {
			log.Error().Err(errUnmarshal).Str("song_data", string(s)).Msg("error unmarshalling song")
			continue
		}
		if (songInfo.Duration == 0 && !songInfo.IsLive) || songInfo.Title == "[Deleted video]" || songInfo.Title == "[Private video]" {
			log.Info().Fields(map[string]interface{}{
				"song_title":       songInfo.Title,
				"song_url":         songInfo.URL,
				"song_is_live":     songInfo.IsLive,
				"song_duration":    songInfo.Duration,
				"song_extractor":   songInfo.Extractor,
				"song_webpage_url": songInfo.WebpageURL,
			}).Msg("Skipping invalid playlist song")
			continue
		}
		log.Debug().Msgf("Found song in playlist: %s", songInfo.Title)
		playListSongs = append(playListSongs, songInfo)
	}
	if len(playListSongs) == 0 {
		log.Debug().Msg("No songs found in playlist")
		return nil, errors.New("no songs found in playlist")
	}
	return playListSongs, nil
}
