package lookup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func getDefinition(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type dictionaryResponse struct {
		Definitions []struct {
			Type       string      `json:"type"`
			Definition string      `json:"definition"`
			Example    *string     `json:"example"`
			ImageURL   string      `json:"image_url"`
			Emoji      interface{} `json:"emoji"`
		} `json:"definitions"`
		Word          string `json:"word"`
		Pronunciation string `json:"pronunciation"`
	}

	if len(args) == 0 {
		instance.SendErrorEmbed("You must provide a word to lookup", "No word provided",
			message.ChannelID)
		return
	}

	url := fmt.Sprintf("https://owlbot.info/api/v4/dictionary/%s", args[0])

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to create GET request for dictionary.")
		instance.SendErrorEmbed("Unable to lookup word", err.Error(),
			message.ChannelID)
		return
	}
	req.Header.Add("Authorization", "Token 2caaf1f54e8c0d10f7a345e6af45aa8c7beeeb50")
	resp, err := instance.HttpClient.Do(req)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to lookup word.")
		instance.SendErrorEmbed("Unable to lookup word.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.WithError(err).Error("Unable to close response body.")
		}
	}()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to read response body of dictionary lookup.")
		instance.SendErrorEmbed("Unable to parse JSON from dictionary API.", err.Error(), message.ChannelID)
		return
	}
	if string(bodyBytes) == `[{"message":"No definition :("}]` {
		instance.SendErrorEmbed("No definition was found", fmt.Sprintf("Definition for %s was not found", args[0]), message.ChannelID)
		return
	}
	jsonDecoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	respJSON := dictionaryResponse{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).WithField("Body", string(bodyBytes)).Error("Unable to parse JSON from dictionary API.")
		instance.SendErrorEmbed("Unable to parse JSON from dictionary API.", err.Error(), message.ChannelID)
		return
	}

	if len(respJSON.Definitions) == 0 {
		instance.SendErrorEmbed("No definitions found", "No definitions found", message.ChannelID)
		return
	}

	embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 28804).
		AddField("Word", args[0], false).
		AddField("Description", respJSON.Definitions[0].Definition, false).
		SetImage(respJSON.Definitions[0].ImageURL).
		MessageEmbed
	instance.SendEmbedMessage(embedmsg, message.ChannelID, "Unable to send dictionary message.")
}
