package music

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize/english"
	"thalassa_discord/pkg/discord"
	"time"
)

func songLeft(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	instance.MusicData.RLock()
	defer instance.MusicData.RUnlock()

	if !instance.MusicData.SongPlaying {
		_, err := instance.Session.ChannelMessageSend(message.ChannelID, "No song is currently playing.")
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to send channel message.")
		}
		return
	}
	songEstimatedEnd := instance.MusicData.SongStarted.Add(time.Duration(instance.MusicData.SongDurationSeconds) * time.Second)
	left := songEstimatedEnd.Sub(time.Now())

	// Convert to hours, minutes, seconds.
	hours := int64(left.Hours())
	minutes := int64(left.Minutes()) % 60
	seconds := int64(left.Seconds()) % 60

	var wordSlice []string

	if hours > 0 {
		wordSlice = append(wordSlice, english.Plural(int(hours), "hour", "hours"))
	}

	if minutes > 0 {
		wordSlice = append(wordSlice, english.Plural(int(minutes), "minute", "minutes"))
	}

	if seconds > 0 {
		wordSlice = append(wordSlice, english.Plural(int(seconds), "second", "seconds"))
	}

	friendlyString := fmt.Sprintf("%s left in the current song.", english.WordSeries(wordSlice, "and"))
	_, err := instance.Session.ChannelMessageSend(message.ChannelID, friendlyString)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to send channel message.")
	}
}
