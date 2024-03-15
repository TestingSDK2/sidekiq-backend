package member

import (
	"context"
	"database/sql"
	"strconv"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"

	"github.com/pkg/errors"
)

func GetAssignedMemberInfo(task map[string]interface{}, profileService peoplerpc.AccountServiceClient) (*peoplerpc.ConciseProfileReply, error) {
	if assignedToID, ok := task["assignedToID"]; ok {
		switch assignedToID := assignedToID.(type) {
		case string:
			if assignedToID != "" {
				return fetchProfile(profileService, assignedToID)
			}
		case float64, float32, int, int64, int32:
			return fetchProfile(profileService, assignedToID)
		}
	}

	return nil, nil
}

func GetReporterInfo(task map[string]interface{}, profileService peoplerpc.AccountServiceClient) (*peoplerpc.ConciseProfileReply, error) {
	if reporter, ok := task["reporter"]; ok {
		switch reporter := reporter.(type) {
		case string:
			if reporter != "" {
				return fetchProfile(profileService, reporter)
			}
		case float64, float32, int, int64, int32:
			return fetchProfile(profileService, reporter)
		}
	}

	return nil, nil
}

func fetchProfile(profileService peoplerpc.AccountServiceClient, reqID interface{}) (*peoplerpc.ConciseProfileReply, error) {
	profileID, err := getProfileID(reqID)
	if err != nil {
		return nil, err
	}

	// assignedMember, err := profileService.FetchConciseProfile(profileID)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
	assignedMember, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "unable to fetch basic info")
	}

	return assignedMember, nil
}

func getProfileID(profileID interface{}) (int, error) {
	switch profileID := profileID.(type) {
	case string:
		return strconv.Atoi(profileID)
	case float64:
		return int(profileID), nil
	case float32:
		return int(profileID), nil
	case int:
		return profileID, nil
	case int64:
		return int(profileID), nil
	case int32:
		return int(profileID), nil
	default:
		return 0, errors.New("invalid profileID type")
	}
}
