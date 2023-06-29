package api

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	connect_go "github.com/bufbuild/connect-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	thalassav1 "thalassa_discord/gen/go/thalassa/v1"
	"thalassa_discord/gen/go/thalassa/v1/thalassav1connect"
	"thalassa_discord/models"
	"thalassa_discord/pkg/discord"
	"thalassa_discord/pkg/music"
)

type Instance struct {
	ShardInstance          *discord.ShardInstance
	songQueueUpdateStreams map[string]map[string]*songRequestUpdateStream
	Host                   string
	Port                   string
	*sync.RWMutex
}

type songRequestUpdateStream struct {
	*connect_go.ServerStream[thalassav1.SongRequestsUpdateStreamResponse]
	Ctx context.Context
	*sync.Mutex
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

func songRequestUpdateEventToProto(event music.SongQueueEvent) *thalassav1.SongRequestsUpdateEvent {
	protoEvent := &thalassav1.SongRequestsUpdateEvent{}
	switch event.Type {
	case music.SongAdded:
		protoEvent.EventType = thalassav1.SongRequestsUpdateEvent_SONG_REQUEST_ADDED
	case music.SongPlaying:
		protoEvent.EventType = thalassav1.SongRequestsUpdateEvent_SONG_REQUEST_PLAYING
	case music.SongFinished:
		protoEvent.EventType = thalassav1.SongRequestsUpdateEvent_SONG_REQUEST_FINISHED
	case music.SongSkipped:
		protoEvent.EventType = thalassav1.SongRequestsUpdateEvent_SONG_REQUEST_SKIPPED
	case music.SongSkippedAll:
		protoEvent.EventType = thalassav1.SongRequestsUpdateEvent_SONG_REQUEST_SKIPPED_ALL
	}
	protoEvent.SongRequest = songRequestModelToProto(event.SongRequest)
	return protoEvent
}

func New(shardInstance *discord.ShardInstance, host, port string) *Instance {
	queueEventMap := make(map[string]map[string]*songRequestUpdateStream)
	return &Instance{
		ShardInstance:          shardInstance,
		songQueueUpdateStreams: queueEventMap,
		Host:                   host,
		Port:                   port,
		RWMutex:                &sync.RWMutex{},
	}
}

func (inst *Instance) SongQueueEventUpdate(guildID string, event music.SongQueueEvent) {
	go func() {
		inst.Lock()
		defer inst.Unlock()
		_, exists := inst.songQueueUpdateStreams[guildID]
		if !exists {
			inst.songQueueUpdateStreams[guildID] = make(map[string]*songRequestUpdateStream)
		}
		for _, stream := range inst.songQueueUpdateStreams[guildID] {
			stream.Lock()
			if stream == nil {
				log.Error().Msgf("Nil song request update stream for guild %s", guildID)
				stream.Unlock()
				continue
			}
			if stream.Ctx.Err() != nil {
				stream.Unlock()
				continue
			}
			errSend := stream.Send(&thalassav1.SongRequestsUpdateStreamResponse{
				Event: songRequestUpdateEventToProto(event),
			})
			if errSend != nil {
				log.Error().Err(errSend).Msgf("Error sending song request update stream for guild %s", guildID)
			}
			stream.Unlock()
		}
	}()
}

func (inst *Instance) Start(ctx context.Context) {

	router := chi.NewRouter()
	apiPath, apiHandler := thalassav1connect.NewAPIServiceHandler(inst)
	log.Debug().Msgf("API path: %s", apiPath)

	router.Use(cors.AllowAll().Handler)
	router.Mount(apiPath, apiHandler)

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
