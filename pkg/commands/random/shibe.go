package random

import (
	"encoding/json"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func getRandomShibePicture(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type shibeJSONResponse []string
	resp, err := instance.HttpClient.Get("https://shibe.online/api/shibes")
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get random dog image.")
		instance.SendErrorEmbed("Unable to get random dog image.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := shibeJSONResponse{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to parse JSON from dog API.")
		instance.SendErrorEmbed("Unable to parse JSON from dog API.", err.Error(), message.ChannelID)
		return
	}

	if len(respJSON) <= 0 {
		instance.Log.Error().Err(err).Msg("Unable to parse JSON from dog API.")
		instance.SendErrorEmbed("Unable to parse JSON from dog API.", err.Error(), message.ChannelID)
		return
	}

	shibeImage := respJSON[0]
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, shibeImage)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to send dog message.")
	}
}
