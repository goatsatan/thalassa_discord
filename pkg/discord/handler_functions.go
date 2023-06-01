package discord

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"sync"
	"time"

	"thalassa_discord/models"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func (*guildMemberAdd) checkNewUserForMute(serverInstance *ServerInstance, guildMemberAdd *discordgo.GuildMemberAdd) {
	serverInstance.RLock()
	defer serverInstance.RUnlock()
	if serverInstance.enabledFeatures.moderationMuteRole {
		mutedUser, err := models.MutedMembers(
			qm.Where("user_id = ?", guildMemberAdd.User.ID),
			qm.And("guild_id = ?", guildMemberAdd.GuildID),
		).One(context.TODO(), serverInstance.Db)
		if err != nil {
			if err != sql.ErrNoRows {
				serverInstance.Log.Error().Err(err).Msg("Unable to get muted user from database.")
				return
			}
		}
		if mutedUser != nil {
			if mutedUser.ExpiresAt.Valid {
				if mutedUser.ExpiresAt.Time.Before(time.Now()) {
					_, err := mutedUser.Delete(context.TODO(), serverInstance.Db)
					if err != nil {
						serverInstance.Log.Error().Err(err).Msg("Unable to delete muted user from database.")
						return
					}
				}
			}
			err = serverInstance.MuteUser(guildMemberAdd.User.ID)
			if err != nil {
				serverInstance.Log.Error().Err(err).Msg("Unable to mute user that rejoined server.")
			}
		}
	}
}

func (*guildCreate) startMusicBot(serverInstance *ServerInstance, guildCreate *discordgo.GuildCreate) {
	serverInstance.RLock()
	defer serverInstance.RUnlock()
	if serverInstance.Configuration.MusicEnabled {
		_, err := serverInstance.Session.ChannelVoiceJoin(guildCreate.ID,
			serverInstance.Configuration.MusicVoiceChannelID.String, false, true)
		if err != nil {
			serverInstance.Log.Error().Err(err).Msg("Unable to join voice")
		} else {
			// If there's a song playing currently don't start playing another song.
			serverInstance.MusicData.RLock()
			songPlaying := serverInstance.MusicData.SongPlaying
			serverInstance.MusicData.RUnlock()
			if !songPlaying {
				serverInstance.HandleSong(serverInstance.Configuration.MusicTextChannelID.String)
			}
		}
	}
}

func (*guildCreate) loadOrCreateDiscordGuildFromDatabase(ctx context.Context, db *sql.DB,
	guildCreate *discordgo.GuildCreate,
) (*models.DiscordServer, error) {

	l := log.With().Fields(map[string]interface{}{
		"guild":              guildCreate.Name,
		"guild_id":           guildCreate.ID,
		"guild_owner":        guildCreate.OwnerID,
		"guild_region":       guildCreate.Region,
		"guild_member_count": guildCreate.MemberCount,
	}).Logger()

	serverInfo, err := models.DiscordServers(
		qm.Where("guild_id = ?", guildCreate.ID)).
		One(ctx, db)
	if err != nil {
		if err != sql.ErrNoRows {
			l.Error().Err(err).Msg("Unable to lookup Discord server from database.")
		}

		// No server configuration found create a new one.
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
		err := newServer.Insert(ctx, db, boil.Infer())
		if err != nil {
			l.Error().Err(err).Msg("Unable to insert Discord server into database.")
			return nil, err
		}
		serverInfo = newServer

	}
	return serverInfo, nil
}

func (*guildCreate) createEveryoneRolePermissionsIfNotExist(instance *ServerInstance, guildCreate *discordgo.GuildCreate,
) {
	everyoneRoleID, err := instance.GetEveryoneRoleID()
	if err != nil {
		return
	}
	_, err = models.RolePermissions(
		qm.Where("guild_id = ?", guildCreate.ID),
		qm.And("role_id = ?", everyoneRoleID),
	).One(context.TODO(), instance.Db)
	if err != nil {
		if err != sql.ErrNoRows {
			instance.Log.Error().Err(err).Msg("Unable to query role permissions from database.")
			return
		}
		// Permission for everyone doesn't exist so we're going to create it.
		newPerms := models.RolePermission{
			GuildID:               guildCreate.ID,
			RoleID:                everyoneRoleID,
			PostLinks:             true,
			ModerationMuteMember:  false,
			RollDice:              true,
			FlipCoin:              true,
			RandomImage:           true,
			UseCustomCommands:     true,
			ManageCustomCommands:  false,
			IgnoreCommandThrottle: false,
			PlaySongs:             true,
			PlayLists:             true,
			SkipSongs:             false,
		}

		err := newPerms.Insert(context.TODO(), instance.Db, boil.Infer())
		if err != nil {
			instance.Log.Error().Err(err).Msg("Unable to insert everyone role permissions in database.")
			return
		}
	}
}

func (*guildCreate) createDiscordGuildInstance(ctx context.Context, db *sql.DB, serverInfo *models.DiscordServer,
	dSession *discordgo.Session, guildCreate *discordgo.GuildCreate,
) *ServerInstance {
	serverCtx, serverCtxCancel := context.WithCancel(ctx)
	musicCtx, musicCtxCancel := context.WithCancel(serverCtx)
	skipAllCtx, skipAllCtxCancel := context.WithCancel(serverCtx)

	l := log.With().Fields(map[string]interface{}{
		"guild":              guildCreate.Name,
		"guild_id":           guildCreate.ID,
		"guild_member_count": guildCreate.MemberCount,
		"guild_joined_at":    guildCreate.JoinedAt,
	}).Logger()

	customCommands, err := models.CustomCommands(qm.Where("guild_id = ?", guildCreate.ID)).All(serverCtx, db)
	if err != nil {
		l.Error().Err(err).Msg("Unable to load custom commands.")
	}

	customCommandsMap := make(map[string]string)
	for _, c := range customCommands {
		customCommandsMap[c.CommandName] = c.Message
	}

	permissions, err := models.RolePermissions(
		qm.Where("guild_id = ?", guildCreate.ID),
	).All(context.TODO(), db)
	if err != nil {
		l.Error().Err(err).Msg("Unable to get role permissions for guild.")
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

	setRolePermsMap := make(map[Permission]*setRolePermsAnswer)
	for _, p := range AllPermissions {
		setRolePermsMap[p] = &setRolePermsAnswer{
			PermissionName: p.FriendlyName(),
			Permission:     p,
			Answered:       false,
		}
	}
	sortedPermissions := AllPermissions
	sort.Slice(sortedPermissions, func(i, j int) bool {
		return sortedPermissions[i].FriendlyName() < sortedPermissions[j].FriendlyName()
	})

	enabledBotFeatures := serverFeatures{
		linkRemoval:        serverInfo.LinkRemovalEnabled,
		music:              serverInfo.MusicEnabled,
		customCommands:     serverInfo.CustomCommandsEnabled,
		diceRoll:           serverInfo.DiceRollEnabled,
		throttleCommands:   serverInfo.ThrottleCommandsEnabled,
		welcomeMessage:     serverInfo.WelcomeMessageEnabled,
		moderationMuteRole: serverInfo.ModerationMuteEnabled,
		notifyMeRole:       serverInfo.NotifyMeRoleEnabled,
	}

	newInstance := &ServerInstance{
		GuildID:         guildCreate.Guild.ID,
		Session:         dSession,
		Configuration:   serverInfo,
		Log:             l,
		Db:              db,
		Ctx:             serverCtx,
		CtxCancel:       serverCtxCancel,
		enabledFeatures: enabledBotFeatures,
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
		CommandSetRolePerms: &setRolePerms{
			UserID:                 "",
			RoleIDBeingSet:         "",
			InProgress:             false,
			SortedPermissionsSlice: sortedPermissions,
			PermissionAnswers:      setRolePermsMap,
			Timeout:                time.Time{},
			RWMutex:                sync.RWMutex{},
		},
		CustomCommands: customCommandsMap,
		HttpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}

	return newInstance
}

func (*guildCreate) handleMutedRole(serverInstance *ServerInstance, guildCreate *discordgo.GuildCreate) {
	// Create muted role if it doesn't exist.
	if serverInstance.enabledFeatures.moderationMuteRole {
		_ = serverInstance.addMutedRoleToAllChannels()
	}
}

func (*guildCreate) handleNotifyRole(serverInstance *ServerInstance) {
	if serverInstance.enabledFeatures.notifyMeRole {
		_, _ = serverInstance.GetOrCreateNotifyRole()
	}
}
