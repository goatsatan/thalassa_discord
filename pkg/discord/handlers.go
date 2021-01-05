package discord

import (
	"context"
	"database/sql"
	"net/http"
	"sync"
	"time"

	"thalassa_discord/models"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
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

	customCommands, err := models.CustomCommands(qm.Where("guild_id = ?", guildCreate.ID)).All(context.TODO(), s.db)
	if err != nil {
		s.log.WithError(err).WithField("Guild Name", guildCreate.Name).Error("Unable to load custom commands.")
	}

	customCommandsMap := make(map[string]string)
	for _, c := range customCommands {
		customCommandsMap[c.CommandName] = c.Message
	}

	permissions, err := models.RolePermissions(
		qm.Where("guild_id = ?", guildCreate.ID),
	).All(context.TODO(), s.db)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"Guild": guildCreate.Name,
		}).WithError(err).Error("Unable to get role permissions for guild.")
	}

	rolePermissions := make(map[string]rolePermission)
	for _, permission := range permissions {
		r := rolePermission{
			roleID:                permission.RoleID,
			postLinks:             permission.PostLinks,
			moderationMuteMember:  permission.ModerationMuteMember,
			rollDice:              permission.RollDice,
			flipCoin:              permission.FlipCoin,
			randomImage:           permission.RandomImage,
			useCustomCommand:      permission.UseCustomCommands,
			manageCustomCommand:   permission.ManageCustomCommands,
			ignoreCommandThrottle: permission.IgnoreCommandThrottle,
			playSongs:             permission.PlaySongs,
			playLists:             permission.PlayLists,
			skipSongs:             permission.SkipSongs,
		}
		rolePermissions[permission.RoleID] = r
	}

	newInstance := &ServerInstance{
		Guild:         guildCreate.Guild,
		Session:       dSession,
		Configuration: serverInfo,
		Log:           s.log,
		db:            s.db,
		Ctx:           ctx,
		CtxCancel:     ctxCancel,
		enabledFeatures: serverFeatures{
			linkRemoval:        serverInfo.LinkRemovalEnabled,
			music:              serverInfo.MusicEnabled,
			customCommands:     serverInfo.CustomCommandsEnabled,
			diceRoll:           serverInfo.DiceRollEnabled,
			throttleCommands:   serverInfo.ThrottleCommandsEnabled,
			welcomeMessage:     serverInfo.WelcomeMessageEnabled,
			moderationMuteRole: serverInfo.ModerationMuteEnabled,
			notifyMeRole:       serverInfo.NotifyMeRoleEnabled,
		},
		rolePermissions: rolePermissions,
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
		customCommands: customCommandsMap,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
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

	// Create muted role if it doesn't exist.
	if newInstance.enabledFeatures.moderationMuteRole {
		_ = newInstance.addMutedRoleToAllChannels()
	}
}

func (s *shardInstance) guildMemberAdd(dSession *discordgo.Session, guildMemberAdd *discordgo.GuildMemberAdd) {
	return
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticate bot has access to.
func (s *shardInstance) messageCreate(dSession *discordgo.Session, messageCreate *discordgo.MessageCreate) {
	message := messageCreate.Content
	if len(message) <= 0 {
		return
	}
	s.RLock()
	serverInstance, _ := s.serverInstances[messageCreate.GuildID]
	s.RUnlock()
	commandFound, commandName, args := s.parseMessageForCommand(messageCreate.Message, serverInstance)
	if commandFound {
		// Custom commands can override built-in commands.
		if !s.handleCustomCommand(commandName, args, messageCreate.Message, serverInstance) {
			s.handleCommand(commandName, args, messageCreate.Message, serverInstance)
		}
	}
	start := time.Now()
	perms, _ := serverInstance.getUserPermissions(messageCreate.Message)
	duration := time.Since(start)
	s.log.Debug(duration)
	s.log.Debugf("%#v\n", perms)
}
