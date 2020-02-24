package discord

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"thalassa_discord/models"

	"github.com/bwmarrin/discordgo"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

func (s *shardInstance) guildCreate(dSession *discordgo.Session, guildCreate *discordgo.GuildCreate) {
	s.log.Infof("Joined Server: %s", guildCreate.Name)
	serverInfo, err := models.DiscordServers(
		qm.Where("guild_id = ?", guildCreate.ID)).
		One(context.Background(), s.db)
	if err != nil {
		if err != sql.ErrNoRows {
			s.log.WithError(err).Error("Unable to lookup Discord server from database.")
		}
		newServer := &models.DiscordServer{
			GuildID:                 guildCreate.ID,
			GuildName:               guildCreate.Name,
			LinkRemovalEnabled:      false,
			MusicEnabled:            false,
			CustomCommandsEnabled:   true,
			DiceRollEnabled:         true,
			PrefixCommand:           "~",
			MusicTextChannelID:      null.String{},
			MusicVoiceChannelID:     null.String{},
			MusicVolume:             0.5,
			AnnounceSongs:           true,
			ThrottleCommandsEnabled: false,
			ThrottleCommandsSeconds: null.Int64{},
			WelcomeMessageEnabled:   false,
			WelcomeMessage:          null.String{},
		}
		err := newServer.Insert(context.Background(), s.db, boil.Infer())
		if err != nil {
			s.log.WithError(err).Error("Unable to insert Discord server into database.")
			return
		}
		serverInfo = newServer
	}
	ctx, ctxCancel := context.WithCancel(context.Background())
	musicCtx, musicCtxCancel := context.WithCancel(context.Background())
	skipAllCtx, skipAllCtxCancel := context.WithCancel(context.Background())
	newInstance := &ServerInstance{
		Guild:         guildCreate.Guild,
		Session:       dSession,
		Configuration: serverInfo,
		Log:           s.log,
		db:            s.db,
		Ctx:           ctx,
		CtxCancel:     ctxCancel,
		MusicData: &musicOpts{
			SongPlaying:         false,
			SongStarted:         time.Time{},
			SongDurationSeconds: 0,
			IsStream:            false,
			Ctx:                 musicCtx,
			CtxCancel:           musicCtxCancel,
			SkipAllCtx:          skipAllCtx,
			SkipAllCtxCancel:    skipAllCtxCancel,
			RWMutex:             sync.RWMutex{},
		},
	}
	s.addServerInstance(guildCreate.ID, newInstance)
	if serverInfo.MusicEnabled {
		_, err := dSession.ChannelVoiceJoin(guildCreate.ID, serverInfo.MusicVoiceChannelID.String, false, true)
		if err != nil {
			s.log.WithError(err).Error("Unable to join voice")
		} else {
			handleSong(newInstance, serverInfo.MusicTextChannelID.String)
		}
	}
}

func (s *shardInstance) guildMemberAdd(dSession *discordgo.Session, guildMemberAdd *discordgo.GuildMemberAdd) {
	return
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func (s *shardInstance) messageCreate(dSession *discordgo.Session, messageCreate *discordgo.MessageCreate) {
	message := messageCreate.Content
	if len(message) <= 0 {
		return
	}
	s.RLock()
	serverInstance, _ := s.serverInstances[messageCreate.GuildID]
	s.RUnlock()
	s.parseMessageForCommand(messageCreate.Message, serverInstance)
}
