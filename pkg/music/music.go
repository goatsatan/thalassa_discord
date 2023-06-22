package music

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ClintonCollins/dca"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func StreamSong(ctx context.Context, link string, log zerolog.Logger, vc *discordgo.VoiceConnection, volume float32) {
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.BufferedFrames = 100
	options.FrameDuration = 20
	options.CompressionLevel = 10
	options.Bitrate = 384
	options.Volume = int(math.Round(float64(volume)))
	log.Debug().Msgf("Streaming song %s", link)

	sCtx, sCtxCancel := context.WithCancel(ctx)
	defer sCtxCancel()

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

	ytdlArgs := []string{
		"--no-progress",
		"--no-call-home",
		"--default-search", "ytsearch",
		"--no-playlist",
		"--no-mtime",
		"--no-warnings",
		"-o", "-",
		"--format",
		"bestaudio*/best",
		"--prefer-ffmpeg",
		"--quiet",
		link,
	}
	cmd := exec.CommandContext(sCtx, "yt-dlp", ytdlArgs...)

	ytdl, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("error getting yt-dlp stdout")
	}
	cmd.Stderr = os.Stderr

	skipped := false

	// Add defer to make sure cleanup is always called.
	defer func(ytCmd *exec.Cmd, ytStdout io.ReadCloser, skippedSong *bool) {
		// Wait for yt-dlp to finish.
		errWait := ytCmd.Wait()
		if errWait != nil && !errors.Is(errWait, context.Canceled) && !skipped {
			log.Error().Err(errWait).Msg("error waiting for yt-dlp")
		}
		// Drain the yt-dlp buffer. This shouldn't need to be drained.
		_, _ = io.Copy(io.Discard, ytStdout)
		// Close yt-dlp stdout. This shouldn't need to be closed.
		_ = ytStdout.Close()
	}(cmd, ytdl, &skipped)

	err = cmd.Start()
	if err != nil {
		log.Error().Err(err).Msg("error starting yt-dlp")
		return
	}

	// Create a new dca encode session.
	encoding, errEncode := dca.EncodeMem(ytdl, options)
	if errEncode != nil {
		log.Error().Err(errEncode).Msg("error encoding song")
		return
	}
	defer encoding.Cleanup()

	// Setup DCA streaming.
	streamChan := make(chan error)
	dca.NewStream(encoding, vc, streamChan)
	select {
	case <-ctx.Done():
		skipped = true
		log.Debug().Msg("song was skipped or program was stopped")
		return
	case errStream := <-streamChan:
		if errStream != nil && errStream != io.EOF {
			log.Error().Err(errStream).Msg("error streaming song")
		}
	}
	log.Debug().Msg("song finished streaming")
}

func GetSongInfo(ctx context.Context, url string) (*Song, error) {
	ytdlCtx, ytdlCtxCancel := context.WithTimeout(ctx, time.Minute*1)
	defer ytdlCtxCancel()
	ytdlpArgs := []string{
		"--dump-json",
		"--no-playlist",
		"--no-progress",
		"--no-warnings",
		"--default-search",
		"ytsearch",
		"--no-call-home",
		"--skip-download",
		url,
	}

	cmd := exec.CommandContext(ytdlCtx, "yt-dlp", ytdlpArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Msg("error getting song info")
		return nil, err
	}
	song := &Song{}
	errUnmarshal := json.Unmarshal(output, song)
	if errUnmarshal != nil {
		log.Error().Err(errUnmarshal).Str("song_data", string(output)).Msg("error unmarshalling song")
		return nil, errUnmarshal
	}
	return song, nil
}

func GetPlaylistInfo(ctx context.Context, url string, shuffle bool) ([]*Song, error) {
	ytdlCtx, ytdlCtxCancel := context.WithTimeout(ctx, time.Minute*5)
	defer ytdlCtxCancel()
	ytdlpArgs := []string{
		"--dump-json",
		"--flat-playlist",
		"--no-progress",
		"--no-warnings",
		"--default-search",
		"ytsearch",
		"--no-call-home",
		"--skip-download",
	}
	if shuffle {
		ytdlpArgs = append(ytdlpArgs, "--playlist-random")
	}
	ytdlpArgs = append(ytdlpArgs, url)

	cmd := exec.CommandContext(ytdlCtx, "yt-dlp", ytdlpArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Msg("error getting playlist info")
		return nil, err
	}
	var playListSongs []*Song
	songs := bytes.Split(output, []byte("\n"))
	for _, song := range songs {
		s := song
		if s == nil || len(s) == 0 {
			continue
		}
		// Ignore hidden videos
		if strings.Contains(string(s), "unavailable video is hidden") {
			continue
		}
		songInfo := &Song{}
		errUnmarshal := json.Unmarshal(s, songInfo)
		if errUnmarshal != nil {
			log.Error().Err(errUnmarshal).Str("song_data", string(s)).Msg("error unmarshalling song")
			continue
		}
		if (songInfo.Duration == 0 && !songInfo.IsLive) || songInfo.Title == "[Deleted video]" || songInfo.Title == "[Private video]" {
			log.Info().Fields(map[string]interface{}{
				"song_title":       songInfo.Title,
				"song_urls":        songInfo.Urls,
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
