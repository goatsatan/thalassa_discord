package random

import (
	"encoding/json"
	"fmt"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func getRandomJoke(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type jokeJSON struct {
		Error    bool   `json:"error"`
		Category string `json:"category"`
		Type     string `json:"type"`
		Setup    string `json:"setup"`
		Delivery string `json:"delivery"`
		Flags    struct {
			Nsfw      bool `json:"nsfw"`
			Religious bool `json:"religious"`
			Political bool `json:"political"`
			Racist    bool `json:"racist"`
			Sexist    bool `json:"sexist"`
		} `json:"flags"`
		ID   int    `json:"id"`
		Lang string `json:"lang"`
	}

	resp, err := instance.HttpClient.Get("https://sv443.net/jokeapi/v2/joke/Any?blacklistFlags=racist&type=twopart")
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get random joke.")
		instance.SendErrorEmbed("Unable to get random joke.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := jokeJSON{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to parse JSON from joke API.")
		instance.SendErrorEmbed("Unable to parse JSON from joke API.", err.Error(), message.ChannelID)
		return
	}

	jokeString := fmt.Sprintf("%s \n %s", respJSON.Setup, respJSON.Delivery)
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, jokeString)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to send joke message.")
	}
}

func getRandomSafeJoke(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type jokeJSON struct {
		Error    bool   `json:"error"`
		Category string `json:"category"`
		Type     string `json:"type"`
		Setup    string `json:"setup"`
		Delivery string `json:"delivery"`
		Flags    struct {
			Nsfw      bool `json:"nsfw"`
			Religious bool `json:"religious"`
			Political bool `json:"political"`
			Racist    bool `json:"racist"`
			Sexist    bool `json:"sexist"`
		} `json:"flags"`
		ID   int    `json:"id"`
		Lang string `json:"lang"`
	}

	resp, err := instance.HttpClient.Get("https://sv443.net/jokeapi/v2/joke/Any?blacklistFlags=nsfw,religious,political,racist,sexist&type=twopart")
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get random joke.")
		instance.SendErrorEmbed("Unable to get random joke.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to close response body.")
		}
	}()
	jsonDecoder := json.NewDecoder(resp.Body)
	respJSON := jokeJSON{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to parse JSON from joke API.")
		instance.SendErrorEmbed("Unable to parse JSON from joke API.", err.Error(), message.ChannelID)
		return
	}

	jokeString := fmt.Sprintf("%s \n %s", respJSON.Setup, respJSON.Delivery)
	_, err = instance.Session.ChannelMessageSend(message.ChannelID, jokeString)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to send joke message.")
	}
}
