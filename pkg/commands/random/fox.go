package random

import (
	"encoding/json"

	"github.com/bwmarrin/discordgo"
	"thalassa_discord/pkg/discord"
)

func getRandomFoxPicture(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type foxJSONResponse struct {
		Image string `json:"image"`
		Link  string `json:"link"`
	}

	resp, err := instance.HttpClient.Get("https://randomfox.ca/floof/")
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get random fox image.")
		instance.SendErrorEmbed("Unable to get random fox image.", err.Error(), message.ChannelID)
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
		instance.SendErrorEmbed("Unable to parse JSON from fox API.", err.Error(), message.ChannelID)
		return
	}

	dogImage := respJSON.Image
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, dogImage)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send fox message.")
	}
}
