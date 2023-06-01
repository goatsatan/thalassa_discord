package discord

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"thalassa_discord/models"

	_ "github.com/lib/pq"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
)

type BotConfiguration struct {
	BotToken     string
	ClientID     string
	ClientSecret string
	DBName       string
	DBUser       string
	DBPassword   string
	DBHost       string
	DBPort       string
	DBSSL        string
}

type handlers struct {
	guildCreate    *guildCreate
	guildMemberAdd *guildMemberAdd
	messageCreate  *messageCreate
}

type ShardInstance struct {
	CloseSignal     chan os.Signal
	directory       string
	ServerInstances map[string]*ServerInstance
	Commands        map[string]*Command
	BotConfig       *BotConfiguration
	handlers        handlers
	ctx             context.Context
	ctxCancel       context.CancelFunc
	Db              *sql.DB
	sync.RWMutex
}

type musicOpts struct {
	SongPlaying          bool
	SongStarted          time.Time
	SongDurationSeconds  int
	IsStream             bool
	Ctx                  context.Context
	CtxCancel            context.CancelFunc
	SkipAllCtx           context.Context
	SkipAllCtxCancel     context.CancelFunc
	CurrentSongRequestID int64
	CurrentSongName      string
	sync.RWMutex
}

type serverFeatures struct {
	linkRemoval        bool
	music              bool
	customCommands     bool
	diceRoll           bool
	throttleCommands   bool
	welcomeMessage     bool
	moderationMuteRole bool
	notifyMeRole       bool
}

type rolePermission struct {
	roleID                string
	postLinks             bool
	moderationMuteMember  bool
	rollDice              bool
	flipCoin              bool
	randomImage           bool
	useCustomCommand      bool
	manageCustomCommand   bool
	ignoreCommandThrottle bool
	playSongs             bool
	playLists             bool
	skipSongs             bool
}

type setRolePermsAnswer struct {
	PermissionName string
	Permission     Permission
	Value          bool
	Answered       bool
}

type setRolePerms struct {
	UserID                 string
	RoleIDBeingSet         string
	InProgress             bool
	SortedPermissionsSlice []Permission
	PermissionAnswers      map[Permission]*setRolePermsAnswer
	Timeout                time.Time
	sync.RWMutex
}

type ServerInstance struct {
	GuildID             string
	Session             *discordgo.Session
	Log                 zerolog.Logger
	Configuration       *models.DiscordServer
	MusicData           *musicOpts
	Ctx                 context.Context
	CtxCancel           context.CancelFunc
	enabledFeatures     serverFeatures
	rolePermissions     map[string]rolePermission
	CommandSetRolePerms *setRolePerms
	Db                  *sql.DB
	HttpClient          *http.Client
	CustomCommands      map[string]string
	sync.RWMutex
}

func setupLog() {
	debug := false
	flag.BoolVar(&debug, "debug", false, "sets log level to debug")
	flag.Parse()
	log.Logger = log.With().Caller().Logger()
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
}

func connectDB(config *BotConfiguration) *sql.DB {
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s dbname=%s sslmode=%s host=%s port=%s password=%s",
		config.DBUser, config.DBName, config.DBSSL, config.DBHost, config.DBPort, config.DBPassword))
	if err != nil {
		log.Fatal().Err(err).Msg("Error connecting to database")
	}
	err = db.Ping()
	if err != nil {
		log.Fatal().Err(err).Msg("Error pinging database")
	}
	return db
}

func NewInstance(ctx context.Context) (*ShardInstance, error) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	setupLog()
	d, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	botConfig := getBotConfiguration()
	db := connectDB(botConfig)
	shardCtx, shardCtxCancel := context.WithCancel(ctx)
	return &ShardInstance{
		CloseSignal: make(chan os.Signal, 1),
		directory:   d,
		ctx:         shardCtx,
		ctxCancel:   shardCtxCancel,
		BotConfig:   botConfig,
		Db:          db,
		handlers: handlers{
			guildCreate:    &guildCreate{},
			guildMemberAdd: &guildMemberAdd{},
			messageCreate:  &messageCreate{},
		},
		Commands:        make(map[string]*Command),
		ServerInstances: make(map[string]*ServerInstance),
	}, nil
}

func getBotConfiguration() *BotConfiguration {
	newConfig := BotConfiguration{}
	currentDirectory, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting current directory")
	}
	configFile, err := os.ReadFile(currentDirectory + "/config.toml")
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading config file")
	}
	_, err = toml.Decode(string(configFile), &newConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Error decoding config file")
	}
	return &newConfig
}

func (s *ShardInstance) gracefulShutdown() {
	s.RLock()
	wg := &sync.WaitGroup{}
	for _, serverInstance := range s.ServerInstances {
		serverInstance.Session.Lock()
		if len(serverInstance.Session.VoiceConnections) > 0 {
			for _, vc := range serverInstance.Session.VoiceConnections {
				wg.Add(1)
				go func(wg *sync.WaitGroup, vc *discordgo.VoiceConnection) {
					defer wg.Done()
					vc.Close()
				}(wg, vc)
			}
		}
		serverInstance.Session.Unlock()
		err := serverInstance.Session.Close()
		if err != nil {
			log.Err(err).Msg("Unable to close bot session.")
		}
	}
	s.ctxCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	go func(wg *sync.WaitGroup, cancel context.CancelFunc) {
		wg.Wait()
		cancel()
	}(wg, cancel)
	<-ctx.Done()
	s.RUnlock()
}

func (s *ShardInstance) Start() {
	signal.Notify(s.CloseSignal, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + s.BotConfig.BotToken)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	logLevel := os.Getenv("THALASSA_LOG_LEVEL")
	if strings.ToLower(logLevel) == "debug" {
		log.Debug().Msg("Setting log level to debug.")
		dg.LogLevel = discordgo.LogDebug
	}

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAll)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(s.messageCreate)
	dg.AddHandler(s.guildCreate)
	dg.AddHandler(s.guildMemberAdd)
	dg.AddHandler(s.guildMemberUpdate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		log.Error().Err(err).Msg("error opening connection")
		return
	}

	log.Info().Msg("Bot is now running.  Press CTRL-C to exit.")
	// Simple way to keep program running until CTRL-C is pressed.
	<-s.CloseSignal
	s.gracefulShutdown()
}
