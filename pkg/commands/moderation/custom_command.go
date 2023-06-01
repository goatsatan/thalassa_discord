package moderation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"thalassa_discord/models"
	"thalassa_discord/pkg/discord"

	"github.com/bwmarrin/discordgo"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func createCustomCommand(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	if len(args) > 1 {
		commandName := args[0]
		commandMessageSlice := args[1:]
		commandMessage := strings.Join(commandMessageSlice, " ")
		customCommand := models.CustomCommand{
			AddedByUserID: message.Author.ID,
			GuildID:       instance.GuildID,
			CommandName:   commandName,
			Message:       commandMessage,
			CreatedAt:     time.Now(),
			UpdatedAt:     null.TimeFrom(time.Now()),
		}

		err := customCommand.Upsert(context.TODO(), instance.Db, true, []string{"guild_id",
			"command_name"},
			boil.Whitelist("message", "updated_at"), boil.Infer())
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to insert custom command.")
			instance.SendErrorEmbed("Database error trying to create custom command.",
				err.Error(), message.ChannelID)
			return
		}

		instance.Lock()
		instance.CustomCommands[commandName] = commandMessage
		instance.Unlock()

		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 28804).
			AddField("Successfully added custom command", args[0], false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, message.ChannelID,
			"Unable to send custom command created message.")
	} else {
		instance.SendErrorEmbed("Invalid command arguments.",
			"You must provide a command name and a message.", message.ChannelID)
	}
}

func deleteCustomCommand(instance *discord.ServerInstance, message *discordgo.Message, args []string) {
	if len(args) > 0 {
		commandName := args[0]
		customCommand, err := models.CustomCommands(
			qm.Where("command_name = ?", commandName),
			qm.And("guild_id = ?", instance.GuildID),
		).One(context.TODO(), instance.Db)
		if err != nil {
			if err != sql.ErrNoRows {
				instance.SendErrorEmbed("Database error trying to delete custom command.", err.Error(),
					message.ChannelID)
				return
			}
			instance.SendErrorEmbed("Custom command doesn't exist.",
				fmt.Sprintf("The custom command %s doesn't exist.", commandName), message.ChannelID)
			return
		}
		_, err = customCommand.Delete(context.TODO(), instance.Db)
		if err != nil {
			instance.SendErrorEmbed("Database error trying to delete custom command.", err.Error(),
				message.ChannelID)
			return
		}

		instance.Lock()
		delete(instance.CustomCommands, commandName)
		instance.Unlock()

		embedmsg := discord.NewEmbedInfer(instance.Session.State.User.Username, 28804).
			AddField("Successfully deleted custom command", args[0], false).
			MessageEmbed
		instance.SendEmbedMessage(embedmsg, message.ChannelID, "Unable to send custom command deleted message.")
	} else {
		instance.SendErrorEmbed("Invalid command arguments.",
			"You must provide a command name", message.ChannelID)
	}
}
