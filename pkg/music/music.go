package music

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"layeh.com/gopus"
)

const (
	YOUTUBE_EXTRACTOR = "youtube:playlist"
)

type PlaylistSong struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Type  string `json:"_type"`
	IeKey string `json:"ie_key"`
	Title string `json:"title"`
}

type Song struct {
	Abr                  float64                `json:"abr"`
	Acodec               string                 `json:"acodec"`
	AgeLimit             int                    `json:"age_limit"`
	Album                interface{}            `json:"album"`
	Artist               interface{}            `json:"artist"`
	AspectRatio          float64                `json:"aspect_ratio"`
	Asr                  int                    `json:"asr"`
	AudioChannels        int                    `json:"audio_channels"`
	AutomaticCaptions    SongAutomaticCaptions  `json:"automatic_captions"`
	Availability         string                 `json:"availability"`
	AverageRating        interface{}            `json:"average_rating"`
	Categories           []string               `json:"categories"`
	Channel              string                 `json:"channel"`
	ChannelFollowerCount int                    `json:"channel_follower_count"`
	ChannelID            string                 `json:"channel_id"`
	ChannelURL           string                 `json:"channel_url"`
	Chapters             interface{}            `json:"chapters"`
	CommentCount         interface{}            `json:"comment_count"`
	Description          string                 `json:"description"`
	DisplayID            string                 `json:"display_id"`
	Duration             int                    `json:"duration"`
	DurationString       string                 `json:"duration_string"`
	DynamicRange         string                 `json:"dynamic_range"`
	Epoch                int                    `json:"epoch"`
	Ext                  string                 `json:"ext"`
	Extractor            string                 `json:"extractor"`
	ExtractorKey         string                 `json:"extractor_key"`
	Filename             string                 `json:"filename"`
	FilesizeApprox       int                    `json:"filesize_approx"`
	Format               string                 `json:"format"`
	FormatID             string                 `json:"format_id"`
	FormatNote           string                 `json:"format_note"`
	FormatSortFields     []string               `json:"_format_sort_fields"`
	Formats              []SongFormats          `json:"formats"`
	Fps                  int                    `json:"fps"`
	Fulltitle            string                 `json:"fulltitle"`
	HasDrm               interface{}            `json:"_has_drm"`
	Height               int                    `json:"height"`
	ID                   string                 `json:"id"`
	IsLive               bool                   `json:"is_live"`
	Language             interface{}            `json:"language"`
	LikeCount            int                    `json:"like_count"`
	LiveStatus           string                 `json:"live_status"`
	OriginalURL          string                 `json:"original_url"`
	PlayableInEmbed      bool                   `json:"playable_in_embed"`
	Playlist             interface{}            `json:"playlist"`
	PlaylistIndex        interface{}            `json:"playlist_index"`
	Protocol             string                 `json:"protocol"`
	ReleaseTimestamp     interface{}            `json:"release_timestamp"`
	RequestedFormats     []SongRequestedFormats `json:"requested_formats"`
	RequestedSubtitles   interface{}            `json:"requested_subtitles"`
	Resolution           string                 `json:"resolution"`
	StretchedRatio       interface{}            `json:"stretched_ratio"`
	Subtitles            SongSubtitles          `json:"subtitles"`
	Tags                 []interface{}          `json:"tags"`
	Tbr                  float64                `json:"tbr"`
	Thumbnail            string                 `json:"thumbnail"`
	Thumbnails           []SongThumbnails       `json:"thumbnails"`
	Title                string                 `json:"title"`
	Track                interface{}            `json:"track"`
	Type                 string                 `json:"_type"`
	UploadDate           string                 `json:"upload_date"`
	Uploader             string                 `json:"uploader"`
	UploaderID           string                 `json:"uploader_id"`
	UploaderURL          string                 `json:"uploader_url"`
	Urls                 string                 `json:"urls"`
	Vbr                  float64                `json:"vbr"`
	Vcodec               string                 `json:"vcodec"`
	Version              SongVersion            `json:"_version"`
	ViewCount            int                    `json:"view_count"`
	WasLive              bool                   `json:"was_live"`
	WebpageURL           string                 `json:"webpage_url"`
	WebpageURLBasename   string                 `json:"webpage_url_basename"`
	WebpageURLDomain     string                 `json:"webpage_url_domain"`
	Width                int                    `json:"width"`
}

type SongVersion struct {
	Version        string      `json:"version"`
	CurrentGitHead interface{} `json:"current_git_head"`
	ReleaseGitHead string      `json:"release_git_head"`
	Repository     string      `json:"repository"`
}

type SongDownloaderOptions struct {
	HTTPChunkSize int `json:"http_chunk_size"`
}

type SongHTTPHeaders struct {
	Accept         string `json:"Accept"`
	AcceptEncoding string `json:"Accept-Encoding"`
	AcceptLanguage string `json:"Accept-Language"`
	UserAgent      string `json:"User-Agent"`
	AcceptCharset  string `json:"Accept-Charset"`
}

type SongFormats struct {
	Abr                float64               `json:"abr,omitempty"`
	Acodec             string                `json:"acodec"`
	AspectRatio        float64               `json:"aspect_ratio"`
	Asr                int                   `json:"asr,omitempty"`
	AudioChannels      int                   `json:"audio_channels,omitempty"`
	AudioExt           string                `json:"audio_ext"`
	Columns            int                   `json:"columns,omitempty"`
	Container          string                `json:"container,omitempty"`
	DownloaderOptions  SongDownloaderOptions `json:"downloader_options,omitempty"`
	DynamicRange       interface{}           `json:"dynamic_range,omitempty"`
	Ext                string                `json:"ext"`
	Filesize           int                   `json:"filesize,omitempty"`
	FilesizeApprox     int                   `json:"filesize_approx,omitempty"`
	Format             string                `json:"format"`
	FormatID           string                `json:"format_id"`
	FormatNote         string                `json:"format_note"`
	Fps                float64               `json:"fps"`
	Fragments          []SongFragments       `json:"fragments,omitempty"`
	HTTPHeaders        SongHTTPHeaders       `json:"http_headers"`
	HasDrm             bool                  `json:"has_drm,omitempty"`
	Height             int                   `json:"height"`
	Language           interface{}           `json:"language,omitempty"`
	LanguagePreference int                   `json:"language_preference,omitempty"`
	Preference         interface{}           `json:"preference,omitempty"`
	Protocol           string                `json:"protocol"`
	Quality            float64               `json:"quality,omitempty"`
	Resolution         string                `json:"resolution"`
	Rows               int                   `json:"rows,omitempty"`
	SourcePreference   int                   `json:"source_preference,omitempty"`
	Tbr                float64               `json:"tbr,omitempty"`
	URL                string                `json:"url"`
	Vbr                float64               `json:"vbr,omitempty"`
	Vcodec             string                `json:"vcodec"`
	VideoExt           string                `json:"video_ext"`
	Width              int                   `json:"width"`
}

type SongFragments struct {
	URL      string  `json:"url"`
	Duration float64 `json:"duration"`
}

type SongSubtitles struct {
}

type SongAutomaticCaptions struct {
}

type SongRequestedFormats struct {
	Abr                float64               `json:"abr,omitempty"`
	Acodec             string                `json:"acodec"`
	AspectRatio        float64               `json:"aspect_ratio"`
	Asr                interface{}           `json:"asr"`
	AudioChannels      interface{}           `json:"audio_channels"`
	AudioExt           string                `json:"audio_ext"`
	Container          string                `json:"container"`
	DownloaderOptions  SongDownloaderOptions `json:"downloader_options"`
	DynamicRange       string                `json:"dynamic_range"`
	Ext                string                `json:"ext"`
	Filesize           int                   `json:"filesize"`
	Format             string                `json:"format"`
	FormatID           string                `json:"format_id"`
	FormatNote         string                `json:"format_note"`
	Fps                int                   `json:"fps"`
	HTTPHeaders        SongHTTPHeaders       `json:"http_headers"`
	HasDrm             bool                  `json:"has_drm"`
	Height             int                   `json:"height"`
	Language           interface{}           `json:"language"`
	LanguagePreference int                   `json:"language_preference"`
	Preference         interface{}           `json:"preference"`
	Protocol           string                `json:"protocol"`
	Quality            float64               `json:"quality"`
	Resolution         string                `json:"resolution"`
	SourcePreference   int                   `json:"source_preference"`
	Tbr                float64               `json:"tbr"`
	URL                string                `json:"url"`
	Vbr                float64               `json:"vbr,omitempty"`
	Vcodec             string                `json:"vcodec"`
	VideoExt           string                `json:"video_ext"`
	Width              int                   `json:"width"`
}

type SongChapters struct {
	EndTime   float64 `json:"end_time"`
	Title     string  `json:"title"`
	StartTime float64 `json:"start_time"`
}

type SongThumbnails struct {
	Height     int    `json:"height,omitempty"`
	ID         string `json:"id"`
	Preference int    `json:"preference"`
	Resolution string `json:"resolution,omitempty"`
	URL        string `json:"url"`
	Width      int    `json:"width,omitempty"`
}

type Playlist struct {
	Type               string            `json:"_type"`
	WebpageURLBasename string            `json:"webpage_url_basename"`
	ExtractorKey       string            `json:"extractor_key"`
	ID                 string            `json:"id"`
	WebpageURL         string            `json:"webpage_url"`
	Extractor          string            `json:"extractor"`
	Title              string            `json:"title"`
	Entries            []PlaylistEntries `json:"entries"`
}
type PlaylistEntries struct {
	Type  string `json:"_type"`
	IeKey string `json:"ie_key"`
	ID    string `json:"id"`
	URL   string `json:"url"`
}

// handleSongProcessCleanup will wait for a process to end so it can be cleaned up. Will cancel error checking if
// passed context is cancelled. Will log an error with your message if there's an error without context being
// cancelled.
func handleSongProcessCleanup(ctx context.Context, c *exec.Cmd, log *logrus.Logger, errorMessage string) {
	select {
	case <-ctx.Done():
		// Song was skipped or we're shutting down so the exit code will be 1 and return an error no matter what.
		_ = c.Wait()
	default:
		err := c.Wait()
		if err != nil {
			log.WithError(err).Error(errorMessage)
		}
	}
}

func StreamSong(ctx context.Context, link string, log *logrus.Logger, vc *discordgo.VoiceConnection, volume float32) {
	// TODO remove log.fatal
	cmd := exec.CommandContext(ctx, "yt-dlp", "--no-progress", "--no-call-home", "--default-search", "ytsearch",
		"--no-playlist", "--no-mtime", "-o", "-", "--format", "bestaudio/worstvideo/best", "--prefer-ffmpeg", "--quiet",
		link)
	run := exec.CommandContext(ctx, "ffmpeg", "-i", "-", "-vn", "-acodec", "pcm_s16le", "-f", "s16le", "-ar", "48000",
		"-af", fmt.Sprintf("volume=%f", volume), "-ac", "2", "pipe:1")
	ytdl, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	r := bufio.NewReaderSize(ytdl, 16384*8)
	run.Stdin = r
	ffmpegout, err := run.StdoutPipe()
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	// Cleanup process after the process ends.
	defer handleSongProcessCleanup(ctx, cmd, log, "Unable to close ffmpeg process.")

	// run.Stderr = os.Stderr
	err = run.Start()
	if err != nil {
		log.Fatal(err)
	}
	// Cleanup process after the process ends.
	defer handleSongProcessCleanup(ctx, run, log, "Unable to close ffmpeg process.")

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)
	err = vc.Speaking(true)
	if err != nil {
		log.WithError(err).Error("Unable to set voice connection to speaking.")
	}
	defer func() {
		err := vc.Speaking(false)
		if err != nil {
			log.WithError(err).Error("Unable to set voice connection to not speaking.")
		}
	}()
	opusEncoder, err := gopus.NewEncoder(48000, 2, gopus.Audio)

	if err != nil {
		fmt.Println("NewEncoder Error:", err)
		return
	}
	for {
		// Increase audiobuf for fun.
		audiobuf := make([]int16, 960*2)
		err = binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			// fmt.Println("EOF")
			break
		}
		if err != nil {
			fmt.Println("error reading from ffmpeg stdout :", err)
			break
		}
		select {
		default:
			select {
			case <-ctx.Done():
				return
			default:
				opus, err := opusEncoder.Encode(audiobuf, 960, 3840)
				if err != nil {
					fmt.Println("Encoding Error:", err)
					return
				}
				if vc.OpusSend == nil {
					fmt.Printf("Discordgo not ready for opus packets.\n")
					return
				}
				vc.OpusSend <- opus
			}
		}
	}
}

func GetSongInfo(ctx context.Context, url string) (*Song, error) {
	song := &Song{}
	cmd := exec.CommandContext(ctx, "yt-dlp", "--simulate", "--print-json", "--no-progress", "--no-call-home",
		"--default-search", "ytsearch", "--no-playlist", "--no-mtime", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return song, err
	}
	err = json.Unmarshal(output, song)
	if err != nil {
		return song, err
	}
	return song, nil
}

func GetPlaylistInfo(ctx context.Context, url string) ([]*PlaylistSong, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp", "--simulate", "--dump-json", "--no-progress", "--no-call-home",
		"--default-search", "ytsearch", "--flat-playlist", "--no-mtime", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	var playlistSongs []*PlaylistSong
	outputString := string(output[:])
	log.Println(outputString)
	songSplit := strings.Split(outputString, "\n")
	for _, songString := range songSplit {
		if songString != "" {
			song := &PlaylistSong{}
			err := json.Unmarshal([]byte(songString), song)
			if err == nil {
				if song.Type == "url" {
					switch song.IeKey {
					case "Youtube":
						song.URL = fmt.Sprintf("https://youtu.be/%s", song.URL)
					}
				}
				playlistSongs = append(playlistSongs, song)
			}
		}
	}
	return playlistSongs, nil
}
