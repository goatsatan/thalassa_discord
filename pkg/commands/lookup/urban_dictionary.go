package lookup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func urbanDictionary(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type dictionaryJSON struct {
		List []struct {
			Definition  string    `json:"definition"`
			Permalink   string    `json:"permalink"`
			ThumbsUp    int       `json:"thumbs_up"`
			SoundUrls   []string  `json:"sound_urls"`
			Author      string    `json:"author"`
			Word        string    `json:"word"`
			Defid       int       `json:"defid"`
			CurrentVote string    `json:"current_vote"`
			WrittenOn   time.Time `json:"written_on"`
			Example     string    `json:"example"`
			ThumbsDown  int       `json:"thumbs_down"`
		} `json:"list"`
	}

	if len(args) == 0 {
		instance.SendErrorEmbed("Invalid word", "You must provide a word.", message.ChannelID)
		return
	}

	url := fmt.Sprintf("https://mashape-community-urban-dictionary.p.rapidapi.com/define?term=%s", args[0])

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to create GET request for dictionary.")
		instance.SendErrorEmbed("Unable to lookup word", err.Error(),
			message.ChannelID)
		return
	}
	req.Header.Add("x-rapidapi-host", "mashape-community-urban-dictionary.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", "cedd53e93amsh465918faf691d51p16006fjsnbf8ea937fa69")
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
	jsonDecoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	respJSON := dictionaryJSON{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		instance.Log.WithError(err).Error("Unable to parse JSON from dictionary lookup.")
		instance.SendErrorEmbed("Unable to parse JSON from dictionary lookup.", err.Error(), message.ChannelID)
		return
	}

	for i, definition := range respJSON.List {
		if i >= 3 {
			break
		}
		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, discord.AQUA).
			AddField("Definition", definition.Definition, false).
			AddField("Example", definition.Example, false).
			MessageEmbed
		embedmsg.Title = fmt.Sprintf("%s definition %d", definition.Word, i+1)
		embedmsg.URL = definition.Permalink
		instance.SendEmbedMessage(embedmsg, message.ChannelID, "Unable to send urban dictionary lookup message.")
	}
}
