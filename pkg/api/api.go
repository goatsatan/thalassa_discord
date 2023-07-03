package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/olahol/melody"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	thalassav1 "thalassa_discord/gen/go/thalassa/v1"
	"thalassa_discord/gen/go/thalassa/v1/thalassav1connect"
	"thalassa_discord/models"
	"thalassa_discord/pkg/discord"
	"thalassa_discord/pkg/music"
)

type Instance struct {
	ShardInstance *discord.ShardInstance
	Host          string
	Port          string
	mSockets      *melody.Melody
	*sync.RWMutex
}

func songRequestModelToProto(songRequestModel *models.SongRequest) *thalassav1.SongRequest {
	return &thalassav1.SongRequest{
		SongName:          songRequestModel.SongName,
		RequestedByUserId: songRequestModel.RequestedByUserID,
		UsernameAtTime:    songRequestModel.UsernameAtTime,
		GuildNameAtTime:   songRequestModel.GuildNameAtTime,
		RequestedAt:       timestamppb.New(songRequestModel.RequestedAt),
		Played:            songRequestModel.Played,
		PlayedAt:          timestamppb.New(songRequestModel.PlayedAt.Time),
		Id:                songRequestModel.ID,
	}
}

func songModelToProto(songModel *models.Song) *thalassav1.Song {
	return &thalassav1.Song{
		SongName:          songModel.SongName,
		Url:               songModel.URL,
		IsStream:          songModel.IsStream,
		Artist:            songModel.Artist.String,
		Track:             songModel.Track.String,
		Album:             songModel.Album.String,
		ThumbnailUrl:      songModel.ThumbnailURL.String,
		DurationInSeconds: uint32(songModel.DurationInSeconds.Int),
		Description:       songModel.Description.String,
		Platform:          songModel.Platform.String,
		Id:                songModel.ID,
	}
}

func New(shardInstance *discord.ShardInstance, host, port string) *Instance {
	return &Instance{
		ShardInstance: shardInstance,
		Host:          host,
		Port:          port,
		RWMutex:       &sync.RWMutex{},
	}
}

func (inst *Instance) SongQueueEventUpdate(guildID string, event music.SongQueueEvent) {
	go func() {
		eventBytes, errMarshal := json.Marshal(&event)
		if errMarshal != nil {
			log.Error().Err(errMarshal).Msg("Error marshalling song request event.")
			return
		}
		errBroadcast := inst.mSockets.BroadcastFilter(eventBytes, func(session *melody.Session) bool {
			return session.Request.URL.Path == "/ws/"+guildID+"/song_request_events"
		})
		if errBroadcast != nil {
			log.Error().Err(errBroadcast).Msg("Error broadcasting song request event.")
		}
	}()
}

func (inst *Instance) handleWebsocketRequest(w http.ResponseWriter, r *http.Request) {
	urlSplit := strings.Split(r.URL.Path, "/")
	if len(urlSplit) != 4 {
		log.Error().Msg("Invalid websocket URL.")
		http.Error(w, "Invalid websocket URL.", http.StatusBadRequest)
		return
	}
	guildID := urlSplit[2]
	inst.ShardInstance.RLock()
	_, exists := inst.ShardInstance.ServerInstances[guildID]
	inst.ShardInstance.RUnlock()
	if !exists {
		log.Error().Str("guild_id", guildID).Msg("Guild not found.")
		http.Error(w, "Guild does not exist.", http.StatusNotFound)
		return
	}
	err := inst.mSockets.HandleRequest(w, r)
	if err != nil {
		log.Error().Err(err).Msg("Error handling websocket request.")
	}
}

func (inst *Instance) Start(ctx context.Context) {

	router := chi.NewRouter()
	apiPath, apiHandler := thalassav1connect.NewAPIServiceHandler(inst)
	log.Debug().Msgf("API path: %s", apiPath)

	inst.mSockets = melody.New()

	router.Use(cors.AllowAll().Handler)
	router.Mount(apiPath, apiHandler)
	router.HandleFunc("/ws/{guild_id}/song_request_events", inst.handleWebsocketRequest)

	httpServer := http.Server{
		Addr:              net.JoinHostPort(inst.Host, inst.Port),
		Handler:           router,
		ReadTimeout:       time.Second * 15,
		ReadHeaderTimeout: time.Second * 15,
		WriteTimeout:      time.Second * 60,
		IdleTimeout:       time.Hour * 1,
	}

	go func() {
		log.Debug().Msg("Thalassa API started and waiting for connections.")
		errListen := httpServer.ListenAndServe()
		if errListen != nil && errListen != http.ErrServerClosed {
			log.Fatal().Err(errListen).Msg("Unable to start Thalassa API.")
		}
		log.Info().Msg("Thalassa API HTTP server stopped.")
	}()

	go func() {
		<-ctx.Done()
		log.Info().Msg("Thalassa API shutting down...")

		shutdownContext, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
		defer cancelFunc()

		errShutdown := httpServer.Shutdown(shutdownContext)
		if errShutdown != nil {
			log.Error().Err(errShutdown).Msg("Error on http server shutdown.")
		}
	}()
}
