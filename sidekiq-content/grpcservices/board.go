package contentgrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	contentrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"

	// contentrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/proto/content"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
)

type ContentGrpcServer struct {
	contentrpc.BoardServiceServer
	App *app.App
}

// Methods to implement

func (server *ContentGrpcServer) AddBoard(ctx context.Context, req *contentrpc.AddBoardRequest) (*contentrpc.GenericResponse, error) {
	b, err := json.Marshal(req.Board)
	if err != nil {
		return nil, err
	}
	var board model.Board
	err = json.Unmarshal(b, &board)
	if err != nil {
		return nil, err
	}
	res, err := server.App.BoardService.AddBoard(board, int(req.ProfileID))
	if err != nil {
		return nil, err
	}
	ret, err := util.SetGenericResponse(res)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (server *ContentGrpcServer) FetchBoardByID(ctx context.Context, req *contentrpc.FetchBoardByIDRequest) (*contentrpc.GenericResponse, error) {
	fmt.Println("req: ", req)
	res, err := server.App.BoardService.FetchBoardByID(req.BoardID, req.Role)
	// fmt.Println("response received: ", res)
	if err != nil {
		return nil, err
	}
	rpcres, err := util.SetGenericResponse(res)
	if err != nil {
		return nil, err
	}
	return rpcres, nil

}

func (server *ContentGrpcServer) ListBoardInvites(ctx context.Context, req *contentrpc.ProfileIDRequest) (*contentrpc.GenericResponse, error) {
	res, err := server.App.BoardService.ListBoardInvites(int(req.ProfileID))
	if err != nil {
		return nil, err
	}

	genres, err := util.SetGenericResponse(res)
	if err != nil {
		return nil, err
	}
	return genres, nil
}

func (server *ContentGrpcServer) UpdateBoardThingsTags(ctx context.Context, req *contentrpc.UpdateBoardThingsTagsRequest) (*contentrpc.GenericResponse, error) {
	err := server.App.BoardService.UpdateBoardThingsTags(int(req.ProfileID), req.BoardID, req.ThingID, req.Tags)
	if err != nil {
		return nil, err
	}
	genres, err := util.SetGenericResponse(
		map[string]interface{}{
			"data":    nil,
			"message": "Tags updated successfully",
			"status":  1,
		},
	)
	if err != nil {
		return nil, err
	}

	return genres, nil
}

func (server *ContentGrpcServer) FetchBoards(ctx context.Context, req *contentrpc.FetchBoardsRequest) (*contentrpc.GenericResponse, error) {
	res, err := server.App.BoardService.FetchBoards(int(req.ProfileID), req.FetchSubBoards, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}
	genres, err := util.SetGenericResponse(res)
	if err != nil {
		return nil, err
	}

	return genres, nil
}

func ConvertStruct(source interface{}, dest interface{}) interface{} {
	sourceValue := reflect.ValueOf(source)
	destValue := reflect.ValueOf(dest).Elem() // Get the addressable value of the destination struct

	for i := 0; i < sourceValue.NumField(); i++ {
		fieldName := sourceValue.Type().Field(i).Name
		destField := destValue.FieldByName(fieldName)

		if destField.IsValid() && destField.CanSet() {
			destField.Set(sourceValue.Field(i))
		}
	}
	return destValue
}

func (server *ContentGrpcServer) GetBoardPermissionByProfile(ctx context.Context, req *contentrpc.GetBoardPermissionByProfileRequest) (*contentrpc.GetBoardPermissionByProfileResponse, error) {
	// convert contentrpc.Board to model.Board
	boards := make([]model.Board, len(req.Boards))
	for i := 0; i < len(req.Boards); i++ {
		var board model.Board
		res := ConvertStruct(req.Boards[i], board)
		boards[i] = res.(model.Board)
	}
	fmt.Println("input boards")
	fmt.Println(req.Boards[:10])
	fmt.Println()

	fmt.Println("model boards")
	fmt.Println(boards[:10])
	fmt.Println()
	// res, err := server.App.BoardService.GetBoardPermissionByProfile(req.Boards, int(req.ProfileID))
	return nil, nil
}

func (server *ContentGrpcServer) GetProfileTags(ctx context.Context, req *contentrpc.ProfileIDRequest) (*contentrpc.GetProfileTagsResponse, error) {
	res, err := server.App.BoardService.GetProfileTags(int(req.ProfileID))
	if err != nil {
		return nil, err
	}
	grpcres := &contentrpc.GetProfileTagsResponse{Tags: res}
	return grpcres, nil
}

func (server *ContentGrpcServer) GetBoardMembers(ctx context.Context, req *contentrpc.GetBoardMembersRequest) (*contentrpc.GetBoardMembersResponse, error) {
	res, err := server.App.BoardService.GetBoardMembers2(req.BoardId, req.Limit, req.Page, req.Search, req.Role)
	if err != nil {
		return nil, err
	}
	// genres, err := util.SetGenericResponse(res)
	// if err != nil {
	// 	return nil, err
	// }
	response := &contentrpc.GetBoardMembersResponse{}
	response.ProfileIDs = res["data"].([]int32)
	response.Message = res["message"].(string)
	response.Status = int32(res["status"].(int))

	return response, err
}
