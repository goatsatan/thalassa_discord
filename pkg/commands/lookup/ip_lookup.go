package lookup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
)

func ipLookup(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	type ipJSON struct {
		IP                 string  `json:"ip"`
		City               string  `json:"city"`
		Region             string  `json:"region"`
		RegionCode         string  `json:"region_code"`
		Country            string  `json:"country"`
		CountryCode        string  `json:"country_code"`
		CountryCodeIso3    string  `json:"country_code_iso3"`
		CountryCapital     string  `json:"country_capital"`
		CountryTld         string  `json:"country_tld"`
		CountryName        string  `json:"country_name"`
		ContinentCode      string  `json:"continent_code"`
		InEu               bool    `json:"in_eu"`
		Postal             string  `json:"postal"`
		Latitude           float64 `json:"latitude"`
		Longitude          float64 `json:"longitude"`
		Timezone           string  `json:"timezone"`
		UtcOffset          string  `json:"utc_offset"`
		CountryCallingCode string  `json:"country_calling_code"`
		Currency           string  `json:"currency"`
		CurrencyName       string  `json:"currency_name"`
		Languages          string  `json:"languages"`
		CountryArea        float64 `json:"country_area"`
		CountryPopulation  float64 `json:"country_population"`
		Asn                string  `json:"asn"`
		Org                string  `json:"org"`
	}

	type errorJSON struct {
		IP     string `json:"ip"`
		Error  bool   `json:"error"`
		Reason string `json:"reason"`
	}

	if len(args) == 0 {
		instance.SendErrorEmbed("Invalid IP/Hostname.", "You must provide an address.", message.ChannelID)
		return
	}

	url := fmt.Sprintf("https://ipapi.co/%s/json/", args[0])

	resp, err := instance.HttpClient.Get(url)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to get IP information.")
		instance.SendErrorEmbed("Unable to get IP information.", err.Error(), message.ChannelID)
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to close response body.")
		}
	}()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		instance.Log.Error().Err(err).Msg("Unable to read response body of IP lookup.")
		instance.SendErrorEmbed("Unable to parse JSON from IP API.", err.Error(), message.ChannelID)
		return
	}
	jsonDecoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	respJSON := ipJSON{}
	err = jsonDecoder.Decode(&respJSON)
	if err != nil {
		errJSON := errorJSON{}
		errDecoder := json.NewDecoder(bytes.NewReader(bodyBytes))
		err = errDecoder.Decode(&errJSON)
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to parse JSON from IP lookup.")
			instance.SendErrorEmbed("Unable to parse JSON from IP lookup.", err.Error(), message.ChannelID)
			return
		}
		instance.SendErrorEmbed("Unable to lookup IP information", errJSON.Reason, message.ChannelID)
	}

	embedmsg := discord.NewEmbedInfer(instance.Session.State.User, 28804).
		AddField("IP", respJSON.IP, false).
		AddField("City", respJSON.City, true).
		AddField("Region", respJSON.Region, true).
		AddField("Country", respJSON.Country, true).
		AddField("ASN", respJSON.Asn, false).
		AddField("ISP", respJSON.Org, true).
		MessageEmbed
	instance.SendEmbedMessage(embedmsg, message.ChannelID, "Unable to send ip lookup message.")
}
