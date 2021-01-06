package random

import (
	"encoding/json"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func getRandomCatPicture(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type catJSONResponse struct {
		File string `json:"file"`
	}

	resp, err := instance.HttpClient.Get("https://aws.random.cat/meow")
	if err != nil {
		instance.Log.WithError(err).Error("Unable to get random cat image.")
		instance.SendErrorEmbed("Unable to get random cat image.", err.Error(), message.ChannelID)
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
		instance.SendErrorEmbed("Unable to parse JSON from cat API.", err.Error(), message.ChannelID)
		return
	}

	dogImage := respJSON.File
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, dogImage)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to send fox message.")
	}
}
