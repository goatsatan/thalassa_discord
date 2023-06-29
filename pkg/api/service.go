package api

import (
	"context"
	"fmt"
	"sync"

	connect_go "github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"google.golang.org/protobuf/types/known/timestamppb"

	thalassav1 "thalassa_discord/gen/go/thalassa/v1"
	"thalassa_discord/models"
)

func (inst *Instance) GetSongRequests(ctx context.Context, request *connect_go.Request[thalassav1.GetSongRequestsRequest]) (*connect_go.Response[thalassav1.GetSongRequestsResponse], error) {
	limit := 25
	if request.Msg.GetLimit() > 0 && request.Msg.GetLimit() <= 250 {
		limit = int(request.Msg.GetLimit())
	}

	orderBy := "requested_at asc"
	if request.Msg.GetOrderBy() != "" {
		direction := "asc"
		if request.Msg.GetOrderDesc() {
			direction = "desc"
		}
		orderBy = fmt.Sprintf("%s %s", request.Msg.GetOrderBy(), direction)
	}

	offset := 0
	if request.Msg.GetOffset() > 0 {
		offset = int(request.Msg.GetOffset())
	}

	songRequestsModel, errGetModels := models.SongRequests(
		qm.Where("guild_id = ?", request.Msg.GetGuildId()),
		qm.OrderBy(orderBy),
		qm.OrderBy("id asc"),
		qm.Limit(limit),
		qm.Offset(offset),
		qm.And("played = ?", false),
		qm.Load(models.SongRequestRels.Song),
	).All(ctx, inst.ShardInstance.Db)
	if errGetModels != nil {
		log.Error().Err(errGetModels).Msgf("Error getting song requests")
		return nil, connect_go.NewError(connect_go.CodeInternal, errGetModels)
	}
	var songRequestsProto []*thalassav1.SongRequest
	for _, srModel := range songRequestsModel {
		songRequestProto := songRequestModelToProto(srModel)
		songRequestProto.Song = songModelToProto(srModel.R.Song)
		songRequestsProto = append(songRequestsProto, songRequestProto)
	}

	total, errCount := models.SongRequests(
		qm.Where("guild_id = ?", request.Msg.GetGuildId()),
		qm.And("played = ?", false),
	).Count(ctx, inst.ShardInstance.Db)
	if errCount != nil {
		log.Error().Err(errCount).Msgf("Error getting song requests count")
		return nil, connect_go.NewError(connect_go.CodeInternal, errCount)
	}

	response := &thalassav1.GetSongRequestsResponse{
		SongRequests: songRequestsProto,
		Total:        int32(total),
	}

	return connect_go.NewResponse(response), nil
}

func (inst *Instance) GetCurrentSongPlaying(ctx context.Context, request *connect_go.Request[thalassav1.GetCurrentSongPlayingRequest]) (*connect_go.Response[thalassav1.GetCurrentSongPlayingResponse], error) {
	inst.ShardInstance.RLock()
	guild, exists := inst.ShardInstance.ServerInstances[request.Msg.GetGuildId()]
	inst.ShardInstance.RUnlock()
	if !exists {
		return nil, connect_go.NewError(connect_go.CodeNotFound, nil)
	}
	if !guild.MusicData.SongPlaying {
		return nil, connect_go.NewError(connect_go.CodeNotFound, nil)
	}
	songRequestProto := songRequestModelToProto(guild.MusicData.CurrentSongRequest)
	songRequestProto.Song = songModelToProto(guild.MusicData.CurrentSong)
	response := &thalassav1.GetCurrentSongPlayingResponse{
		RequestedAt: songRequestProto.RequestedAt,
		StartedAt:   timestamppb.New(guild.MusicData.SongStarted),
		Song:        songRequestProto.Song,
		RequestedBy: songRequestProto.UsernameAtTime,
		SongRequest: songRequestProto,
	}
	return connect_go.NewResponse(response), nil
}

func (inst *Instance) SongRequestsUpdateStream(ctx context.Context,
	request *connect_go.Request[thalassav1.SongRequestsUpdateStreamRequest],
	response *connect_go.ServerStream[thalassav1.SongRequestsUpdateStreamResponse],
) error {
	guildID := request.Msg.GetGuildId()
	inst.Lock()
	_, exists := inst.songQueueUpdateStreams[guildID]
	if !exists {
		inst.ShardInstance.RLock()
		_, gExists := inst.ShardInstance.ServerInstances[guildID]
		inst.ShardInstance.RUnlock()
		if !gExists {
			log.Error().Msgf("Error getting song request update stream for guild %s", guildID)
			inst.Unlock()
			return connect_go.NewError(connect_go.CodeNotFound, nil)
		}
		inst.songQueueUpdateStreams[guildID] = make(map[string]*songRequestUpdateStream)
	}
	// Create UUID to track stream
	streamID := uuid.New().String()
	inst.songQueueUpdateStreams[guildID][streamID] = &songRequestUpdateStream{
		ServerStream: response,
		Ctx:          ctx,
		Mutex:        &sync.Mutex{},
	}
	inst.Unlock()
	select {
	case <-ctx.Done():
		break
	case <-inst.ShardInstance.Ctx.Done():
		break
	}
	// Remove stream from map
	inst.Lock()
	delete(inst.songQueueUpdateStreams[guildID], streamID)
	inst.Unlock()
	return nil
}
