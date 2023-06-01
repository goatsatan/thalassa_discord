package music

import (
	"context"
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/rs/zerolog"
	"github.com/wader/goutubedl"
	"io"
	"math"
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
	playList, errInfo := goutubedl.New(ctx, url, goutubedl.Options{
		Type: goutubedl.TypePlaylist,
	})
	if errInfo != nil {
		return nil, errInfo
	}
	if playList.Info.Type != "playlist" && playList.Info.Type != "multi_video" {
		return nil, errors.New("not a playlist")
	}
	var playlistSongs []*goutubedl.Info
	for _, song := range playList.Info.Entries {
		playlistSongs = append(playlistSongs, &song)
	}
	return playlistSongs, nil
}
