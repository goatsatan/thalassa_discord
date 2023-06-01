package random

import (
	"encoding/json"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func getRandomDuckPicture(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type duckJSONResponse struct {
		Message string `json:"message"`
		URL     string `json:"url"`
	}
	resp, err := instance.HttpClient.Get("https://random-d.uk/api/v2/random")
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get random duck image.")
		instance.SendErrorEmbed("Unable to get random duck image.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := duckJSONResponse{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to parse JSON from duck API.")
		instance.SendErrorEmbed("Unable to parse JSON from duck API.", err.Error(), message.ChannelID)
		return
	}

	duckImage := respJSON.URL
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, duckImage)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to send duck message.")
	}
}
