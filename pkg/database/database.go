package database

import (
	"context"
	"database/sql"

	"thalassa_discord/models"

	"github.com/volatiletech/sqlboiler/queries/qm"
)

func getGuildConfiguration(db *sql.DB, guildID string) (*models.DiscordServer, error) {
	return models.DiscordServers(
		qm.Where("guild_id = ?", guildID)).
		One(context.Background(), db)
}
