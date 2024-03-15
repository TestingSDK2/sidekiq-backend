package searchgrpc

import (
	"context"
	"fmt"

	// search "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/proto/search"
	searchrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/util"
)

type SearchGrpcServer struct {
	App *app.App
	searchrpc.SearchServiceServer
}

func (server *SearchGrpcServer) UpdateSearchResult(ctx context.Context, req *searchrpc.UpdateSearchResultRequest) (*searchrpc.UpdateSearchResultResponse, error) {
	response := &searchrpc.UpdateSearchResultResponse{}
	fmt.Println("received data")
	fmt.Println(req.Data)

	fmt.Println("search services")
	fmt.Println("app: ", server.App)
	fmt.Println("app.SearchService: ", server.App.SearchService)

	// data
	note, err := util.ConvertAnyToMap(req.Data)
	if err != nil {
		response.Error = &searchrpc.Status{Code: 0, Message: err.Error()}
		return response, err
	}

	fmt.Println()
	fmt.Println("after converting")
	fmt.Println(note)

	err = server.App.SearchService.UpdateSearchResults(note, req.UpdateType)
	if err != nil {
		response.Error = &searchrpc.Status{Code: 0, Message: err.Error()}
		return response, err
	}

	response.Error = &searchrpc.Status{Code: 1}
	return response, nil
}
