package discord

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"thalassa_discord/models"

	_ "github.com/lib/pq"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/onrik/logrus/filename"
	"github.com/sirupsen/logrus"
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
	Log             *logrus.Logger
	ServerInstances map[string]*ServerInstance
	Commands        map[string]*Command
	BotConfig       *BotConfiguration
	handlers        handlers
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
	Log                 *logrus.Logger
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

func setupLog() *logrus.Logger {
	l := logrus.New()
	l.AddHook(filename.NewHook())
	l.Level = logrus.DebugLevel
	return l
}

func connectDB(config *BotConfiguration) *sql.DB {
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s dbname=%s sslmode=%s host=%s port=%s password=%s",
		config.DBUser, config.DBName, config.DBSSL, config.DBHost, config.DBPort, config.DBPassword))
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func NewInstance() (*ShardInstance, error) {
	rand.Seed(time.Now().UTC().UnixNano())
	d, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	botConfig := getBotConfiguration()
	db := connectDB(botConfig)
	return &ShardInstance{
		CloseSignal: make(chan os.Signal, 1),
		directory:   d,
		BotConfig:   botConfig,
		Log:         setupLog(),
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
		log.Fatal(err)
	}
	configFile, err := ioutil.ReadFile(currentDirectory + "/config.toml")
	if err != nil {
		log.Fatal(err)
	}
	_, err = toml.Decode(string(configFile), &newConfig)
	if err != nil {
		log.Fatal(err)
	}
	return &newConfig
}

func (s *ShardInstance) gracefulShutdown() {
	s.RLock()
	for _, serverInstance := range s.ServerInstances {
		serverInstance.Session.Lock()
		if len(serverInstance.Session.VoiceConnections) > 0 {
			for _, vc := range serverInstance.Session.VoiceConnections {
				vc.Close()
			}
		}
		serverInstance.Session.Unlock()
		err := serverInstance.Session.Close()
		if err != nil {
			s.Log.WithError(err).Error("Unable to close bot session.")
		}
	}
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

	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildMembers)

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(s.messageCreate)
	dg.AddHandler(s.guildCreate)
	dg.AddHandler(s.guildMemberAdd)
	dg.AddHandler(s.guildMemberUpdate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	// Simple way to keep program running until CTRL-C is pressed.
	<-s.CloseSignal
	s.gracefulShutdown()
}
