package random

import (
	"encoding/json"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func getRandomDogPicture(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type dogJSONResponse struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}
	resp, err := instance.HttpClient.Get("https://dog.ceo/api/breeds/image/random")
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get random dog image.")
		instance.SendErrorEmbed("Unable to get random dog image.", err.Error(), message.ChannelID)
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
		instance.SendErrorEmbed("Unable to parse JSON from dog API.", err.Error(), message.ChannelID)
		return
	}

	dogImage := respJSON.Message
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, dogImage)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send dog message.")
	}
}
