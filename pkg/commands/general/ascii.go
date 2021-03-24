package general

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/url"

	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
	"github.com/h2non/filetype"
	"github.com/qeesung/image2ascii/convert"
)

func getAsciiFromImage(img image.Image) string {
	converter := convert.NewImageConverter()
	opts := convert.DefaultOptions
	opts.FixedHeight = 20
	opts.FixedWidth = 50
	opts.Colored = false
	return converter.Image2ASCIIString(img, &opts)
}

// decodePngFromReader decode reader to png image.
func decodePngFromReader(imgReader io.Reader) (image.Image, error) {
	i, err := png.Decode(imgReader)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// decodeJpgFromReader decode reader to jpeg image.
func decodeJpgFromReader(imgReader io.Reader) (image.Image, error) {
	i, err := jpeg.Decode(imgReader)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// generateAsciiArtFromImageURL tries a URL for an image. If the mimetype is correct it tried to decode it into an image
// then generates art and sends it to channel.
func generateAsciiArtFromImageURL(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	if len(args) > 0 {
		_, err := url.ParseRequestURI(args[0])
		if err != nil {
			instance.SendErrorEmbed("Invalid command argument.", "You must specify a valid URL.",
				message.ChannelID)
			return
		}
		resp, err := instance.HttpClient.Get(args[0])
		if err != nil {
			instance.SendErrorEmbed("Unable to get image from URL.", err.Error(), message.ChannelID)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		respBody1 := new(bytes.Buffer)
		respBody2 := io.TeeReader(resp.Body, respBody1)

		fType, err := filetype.MatchReader(respBody2)
		if err != nil {
			instance.SendErrorEmbed("Unable to get image type.", err.Error(), message.ChannelID)
			return
		}

		// Discard the rest of respBody2 since we only needed to get the mime type from it. Reading and discarding
		// populates respBody1 to be used later.
		_, err = io.Copy(io.Discard, respBody2)
		if err != nil {
			log.Fatal(err)
		}

		var asciiArt string
		// Check if the file type is correct.
		switch fType.MIME.Value {
		case "image/jpeg":
			img, err := decodeJpgFromReader(respBody1)
			if err != nil {
				instance.SendErrorEmbed("Unable to decode JPEG image.", err.Error(), message.ChannelID)
				return
			}
			asciiArt = getAsciiFromImage(img)
		case "image/png":
			img, err := decodePngFromReader(respBody1)
			if err != nil {
				instance.SendErrorEmbed("Unable to decode PNG image.", err.Error(), message.ChannelID)
				return
			}
			asciiArt = getAsciiFromImage(img)
		default:
			instance.SendErrorEmbed("Invalid image type.", "You must use a JPEG or PNG image.",
				message.ChannelID)
			return
		}
		_, err = instance.Session.ChannelMessageSend(message.ChannelID,
			fmt.Sprintf("```%s```", asciiArt))
		if err != nil {
			instance.Log.WithError(err).Error("Unable to send channel message for ascii art.")
		}
	} else {
		instance.SendErrorEmbed("Invalid command argument.", "You must specify a valid URL.",
			message.ChannelID)
		return
	}
}
