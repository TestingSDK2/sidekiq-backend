package task

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/member"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/consts"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/util"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	notfrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	searchrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (a *api) FetchTasksOfBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")

	taskRes, err := a.taskService.FetchTasksOfBoard(boardID, ctx.Profile, "", []string{}, "", 0, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(taskRes)
	}
	return err
}

func (a *api) AddTask(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var err error

	var payload, task map[string]interface{}

	var post model.Post
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}

	postObjectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return err
	}

	postdata, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}
	post = postdata["data"].(model.Post)

	taskRes, err := a.taskService.AddTask(postObjectID, payload)
	if err != nil {
		return errors.Wrap(err, "unable to add to Task")
	}

	if taskRes["data"] != nil && taskRes["status"].(int) == 1 {

		err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
		if err != nil {
			return errors.Wrap(err, "unable to update tags")
		}

		task = taskRes["data"].(map[string]interface{})
		assigneeIDStr := ""
		if assigneeID, ok := task["assignedToID"]; ok {
			if assigneeID.(string) != "" {
				assigneeIDStr = assigneeID.(string)
			}
		}

		senderId := ctx.Profile

		if assigneeIDStr != "" {
			if postdata["data"] != nil && postdata["status"].(int) == 1 {
				receiverID, err := strconv.Atoi(assigneeIDStr)
				if err != nil {
					return errors.Wrap(err, "unable to parse AssignedToID")
				}

				// err = a.notificationService.NotificationHandler(receiverID, senderId, a.clientMgr, consts.TaskType, task["_id"].(primitive.ObjectID).Hex(), consts.TaskInitiated, "")
				// if err != nil {
				// 	return errors.Wrap(err, "unable send and create notification")
				// }

				// grpc NOTF
				request := notfrpc.NotificationHandlerRequest{
					ReceiverIDs: []int32{int32(receiverID)},
					SenderID:    int32(senderId),
					ThingType:   consts.TaskType,
					ActionType:  consts.TaskInitiated,
					Message:     "",
					ThingID:     task["_id"].(primitive.ObjectID).Hex(),
				}
				_, err = a.repos.NotificationGrpcServiceClient.NotificationHandler(context.TODO(), &request)
				if err != nil {
					return errors.Wrap(err, "unable send and create notification")
				}
			}
		}

		// senderInfo, err = a.profileService.FetchConciseProfile(senderId)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to get profile details")
		// }

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(senderId)}
		senderInfo, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return err
		}

		// add to search results
		task["boardID"] = post.BoardID
		// err = a.searchService.UpdateSearchResults(task, "insert")
		// if err != nil {
		// 	return err
		// }

		reqVal, err := util.MapToMapAny(task)
		if err != nil {
			return err
		}
		in := &searchrpc.UpdateSearchResultRequest{
			Data:       reqVal,
			UpdateType: "insert",
			Args:       "",
		}
		_, err = a.repos.SearchGrpcServiceClient.UpdateSearchResult(context.TODO(), in)
		if err != nil {
			return err
		}

		// activity
		msg := model.ThingActivity{}
		msg.Create(
			primitive.NilObjectID,
			task["postID"].(primitive.ObjectID),
			ctx.Profile,
			strings.ToUpper(consts.TaskType),
			fmt.Sprintf("%s %s updated the task <b>%s</b>", senderInfo.FirstName, senderInfo.LastName, util.GetTitle(task)),
		)
		err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
		if err != nil {
			return errors.Wrap(err, "unable to push to SQS")
		}
	}

	json.NewEncoder(w).Encode(taskRes)

	return err
}

func (a *api) UpdateTask(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID, taskID, postID := ctx.Vars["boardID"], ctx.Vars["taskID"], ctx.Vars["postID"]
	if boardID == "" || taskID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	var payload map[string]interface{}
	var senderInfo *peoplerpc.ConciseProfileReply
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}

	// check if post or board exists
	postdata, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}
	post := postdata["data"].(model.Post)
	senderId, err := strconv.Atoi(string(post.Owner))
	if err != nil {
		return errors.Wrap(err, "unable to parse post.Owner")
	}
	// senderInfo, err = a.profileService.FetchConciseProfile(senderId)
	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(senderId)}
	senderInfo, err = a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	if err != nil {
		return errors.Wrap(err, "unable to get ctx.Profile details")
	}

	taskRes, err := a.taskService.UpdateTask(payload, boardID, postID, taskID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "unable to update Task")
	}
	if taskRes["status"].(int) != 0 {
		if taskRes["data"] != nil {
			var tagsFinal []string
			taskdata := taskRes["data"].(map[string]interface{})

			if tags, ok := taskdata["tags"]; ok {
				if tagsarray, ok := tags.(primitive.A); ok {
					for _, t := range tagsarray {
						tagsFinal = append(tagsFinal, t.(string))
					}
				}
			}

			err = a.boardService.UpdateBoardThingsTags(ctx.Profile, boardID, taskID, tagsFinal)
			if err != nil {
				return errors.Wrap(err, "unable to update Profile Tags")
			}
			err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
			if err != nil {
				return errors.Wrap(err, "unable to update tags")
			}

			assignedMemberInfo, err := member.GetAssignedMemberInfo(taskdata, a.repos.PeopleGrpcServiceClient)
			if err != nil {
				return errors.Wrap(err, "unable to get assignedMemberInfo")
			}
			taskRes["data"].(map[string]interface{})["assignedMemberInfo"] = assignedMemberInfo

			reporterInfo, err := member.GetReporterInfo(taskdata, a.repos.PeopleGrpcServiceClient)
			if err != nil {
				return errors.Wrap(err, "unable to get reporterInfo")
			}
			taskRes["data"].(map[string]interface{})["reporterInfo"] = reporterInfo
		}
	}

	if taskRes["data"] != nil && taskRes["status"].(int) == 1 {
		task := taskRes["data"].(map[string]interface{})
		updatedtaskstatus := ""
		oldtaskStatus := ""

		if taskStatus, ok := payload["taskStatus"]; ok {
			oldtaskStatus = taskStatus.(string)
		}

		if taskStatus, ok := task["taskStatus"]; ok {
			updatedtaskstatus = taskStatus.(string)
		}

		if assignedToID, ok := task["assignedToID"]; ok {
			assignedToIDStr := assignedToID.(string)
			if assignedToIDStr != "" {
				receiverID, err := strconv.Atoi(assignedToIDStr)
				if err != nil {
					return errors.Wrap(err, "unable to parse AssignedToID")
				}
				fmt.Println(receiverID)

				if (assignedToIDStr != "" && updatedtaskstatus == "completed" && oldtaskStatus != "completed") || (assignedToIDStr != "" && updatedtaskstatus == "closed" && oldtaskStatus != "closed") {
					if postdata["data"] != nil && postdata["status"].(int) == 1 {
						message := fmt.Sprintf("%s task status changed from %s to %s", task["title"].(string), oldtaskStatus, updatedtaskstatus)
						fmt.Println(message)
						// err = a.notificationService.NotificationHandler(receiverID, senderId, a.clientMgr, consts.TaskType, taskID, consts.TaskStatusUpdated, message)
						// if err != nil {
						// 	return errors.Wrap(err, "unable send and create notification")
						// }

						// CALL GRPC
						request := notfrpc.NotificationHandlerRequest{
							ReceiverIDs: []int32{int32(receiverID)},
							SenderID:    int32(senderId),
							ThingType:   consts.TaskType,
							ActionType:  consts.TaskInitiated,
							Message:     message,
							ThingID:     taskID,
						}
						_, err = a.repos.NotificationGrpcServiceClient.NotificationHandler(context.TODO(), &request)
						if err != nil {
							return errors.Wrap(err, "unable send and create notification")
						}

					}
				} else if assignedToIDStr != "" { // task "changed"
					if postdata["data"] != nil && postdata["status"].(int) == 1 {
						message := fmt.Sprintf("%s task has beed changed by", task["title"])
						fmt.Println(message)
						// CALL GRPC
						// err = a.notificationService.NotificationHandler(receiverID, senderId, a.clientMgr, consts.TaskType, taskID, consts.TaskUpdated, message)
						// if err != nil {
						// 	return errors.Wrap(err, "unable send and create notification")
						// }

						request := notfrpc.NotificationHandlerRequest{
							ReceiverIDs: []int32{int32(receiverID)},
							SenderID:    int32(senderId),
							ThingType:   consts.TaskType,
							ActionType:  consts.TaskUpdated,
							Message:     message,
							ThingID:     taskID,
						}
						_, err = a.repos.NotificationGrpcServiceClient.NotificationHandler(context.TODO(), &request)
						if err != nil {
							return errors.Wrap(err, "unable send and create notification")
						}
					}
				}
			}
		}

		// add to search results
		task["boardID"] = post.BoardID
		// err = a.searchService.UpdateSearchResults(task, "update")
		// if err != nil {
		// 	return err
		// }

		reqVal, err := util.MapToMapAny(task)
		if err != nil {
			return err
		}
		in := &searchrpc.UpdateSearchResultRequest{
			Data:       reqVal,
			UpdateType: "update",
			Args:       "",
		}
		_, err = a.repos.SearchGrpcServiceClient.UpdateSearchResult(context.TODO(), in)
		if err != nil {
			return err
		}

		// activity
		msg := model.ThingActivity{}
		msg.Create(
			primitive.NilObjectID,
			task["postID"].(primitive.ObjectID),
			ctx.Profile,
			strings.ToUpper(consts.TaskType),
			fmt.Sprintf("%s %s updated the task <b>%s</b>", senderInfo.FirstName, senderInfo.LastName, util.GetTitle(task)),
		)
		err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
		if err != nil {
			return errors.Wrap(err, "unable to push to SQS")
		}
	}

	json.NewEncoder(w).Encode(taskRes)

	return err
}

func (a *api) DeleteTask(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID, taskID, postID := ctx.Vars["boardID"], ctx.Vars["taskID"], ctx.Vars["postID"]
	if boardID == "" || taskID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	task, err := a.taskService.DeleteTask(boardID, taskID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "unable to delete Task")
	}

	if task["status"].(int) == 1 {
		// delete from recentlyAdded, if present
		// _, err = a.recentThingsService.DeleteFromBoardThingsRecent(ctx.Profile, taskID, boardID)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to delete from Board's recent things")
		// }

		// flag from bookmarks, if present
		_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, taskID, time.Now())
		if err != nil {
			fmt.Println(err.Error())
			return errors.Wrap(err, "unable to update flag in bookmark")
		}

		// add to search results
		// err = a.searchService.UpdateSearchResults(nil, "delete", taskID)
		// if err != nil {
		// 	return err
		// }

		in := &searchrpc.UpdateSearchResultRequest{
			Data:       nil,
			UpdateType: "delete",
			Args:       taskID,
		}
		_, err = a.repos.SearchGrpcServiceClient.UpdateSearchResult(context.TODO(), in)
		if err != nil {
			return err
		}

		// log activity
		// cp, err := a.profileService.FetchConciseProfile(ctx.Profile)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to fetch concise profile")
		// }

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
		cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return err
		}

		postObjID, err := primitive.ObjectIDFromHex(postID)
		if err != nil {
			return errors.Wrap(err, "unable to convert string to objectID")
		}

		msg := model.ThingActivity{}
		// activity
		msg.Create(
			primitive.NilObjectID,
			postObjID,
			ctx.Profile,
			strings.ToUpper(consts.TaskType),
			fmt.Sprintf("%s %s deleted the task <b>%s</b>", cp.FirstName, cp.LastName, util.GetTitle(task["data"].(map[string]interface{}))),
		)
		err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
		if err != nil {
			return errors.Wrap(err, "unable to push to SQS")
		}

		json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Task deleted successfully"))
		return nil
	}

	task["data"] = nil
	json.NewEncoder(w).Encode(task)

	return err
}

// GetTaskByID
func (a *api) GetTaskByID(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID, taskID, postID := ctx.Vars["boardID"], ctx.Vars["taskID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" || taskID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var err error

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}

	taskRes, err := a.taskService.GetTaskByID(taskID, ctx.Profile)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(taskRes)
	return nil
}

// GetTaskByID
func (a *api) ActionList(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	sortBy := r.URL.Query().Get("sortBy")
	orderBy := r.URL.Query().Get("orderBy")
	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")
	filterBy := r.URL.Query().Get("filterBy")

	var limitInt, pageInt int
	var err error
	if limit != "" && page != "" {
		pageInt, err = strconv.Atoi(page)
		if err != nil {
			return errors.Wrap(err, "unable to convert string to int")
		}
		limitInt, err = strconv.Atoi(limit)
		if err != nil {
			return errors.Wrap(err, "unable to convert string to int")
		}
	} else {
		limitInt = 10
		pageInt = 1
	}

	taskRes, err := a.taskService.GetActionTask(ctx.Profile, sortBy, orderBy, limitInt, pageInt, filterBy)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(taskRes)
	return nil
}
