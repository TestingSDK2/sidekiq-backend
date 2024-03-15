package note

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/pkg/errors"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	// searchrpc "github.com/sidekiq-search/proto/search"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	searchrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FetchNotesByBoard - fetches note for the boards if the current user is viewer, member, admin or owner
func (a *api) FetchNotesByBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")

	boardId := ctx.Vars["boardID"]
	if boardId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	notes, err := a.noteService.FetchNotesByBoard(boardId, ctx.Profile, "", []string{}, "", 0, limit, page)
	if err == nil {
		json.NewEncoder(w).Encode(notes)
		return nil
	}
	return err
}

// AddNote - insert the text for the boards if the current user is viewer, member, admin or owner
func (a *api) AddNote(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	profileID := ctx.Profile
	if profileID == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	var note interface{}
	var err error

	err = json.NewDecoder(r.Body).Decode(&note)
	if err != nil {
		return errors.Wrap(err, "unable to decode note json")
	}

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.Wrap(err, "unable to convert string to objectID")
	}

	postRes, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}

	noteRes, err := a.noteService.AddNote(postObjID, note)
	if err != nil {
		return errors.Wrap(err, "unable to add Note")
	}

	post := postRes["data"].(model.Post)

	// add to search results
	noteRes["data"].(map[string]interface{})["boardID"] = post.BoardID
	// err = a.searchService.UpdateSearchResults(noteRes["data"].(map[string]interface{}), "insert")
	// if err != nil {
	// 	return err
	// }

	reqVal, err := util.MapToMapAny(noteRes["data"].(map[string]interface{}))
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

	// add to activity
	// cp, _ := a.profileService.FetchConciseProfile(ctx.Profile)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
	cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	msg := model.ThingActivity{}
	msg.Create(
		primitive.NilObjectID,
		noteRes["data"].(map[string]interface{})["postID"].(primitive.ObjectID),
		ctx.Profile,
		strings.ToUpper(consts.NoteType),
		fmt.Sprintf("%s %s added the note <b>%s</b>", cp.FirstName, cp.LastName, util.GetTitle(noteRes["data"].(map[string]interface{}))),
	)
	err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
	if err != nil {
		return errors.Wrap(err, "unable to push to SQS")
	}

	json.NewEncoder(w).Encode(noteRes)
	return err
}

// UpdateNote - update the text for the boards if the current user is viewer, member, admin or owner
func (a *api) UpdateNote(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID, noteID, postID := ctx.Vars["boardID"], ctx.Vars["noteID"], ctx.Vars["postID"]
	if boardID == "" || noteID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var payload map[string]interface{}
	var err error
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}

	// check if post or board exists
	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return err
	}

	note, err := a.noteService.UpdateNote(payload, boardID, postID, noteID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "unable to update Note")
	}
	if note["status"].(int) != 0 {
		// update tags
		// p := model.Profile{ID: ctx.Profile}
		// err = a.profileService.UpdateProfileTags(p, note["data"].(model.Note).Tags)
		// primitive.A to []string

		if note["data"] != nil {
			if note["data"].(map[string]interface{})["tags"] != nil {
				var tags []string
				for _, t := range note["data"].(map[string]interface{})["tags"].(primitive.A) {
					tags = append(tags, t.(string))
				}

				err = a.boardService.UpdateBoardThingsTags(ctx.Profile, boardID, noteID, tags)
				if err != nil {
					return err
				}

				err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
				if err != nil {
					return errors.Wrap(err, "unable to update tags")
				}
			}
		}

		// add to search results
		// err = a.searchService.UpdateSearchResults(note["data"].(map[string]interface{}), "update")
		// if err != nil {
		// 	return err
		// }

		reqVal, err := util.MapToMapAny(note["data"].(map[string]interface{}))
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

		// add to activity (need to re-discuss)
		// cp, _ := a.profileService.FetchConciseProfile(ctx.Profile)

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
		cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return err
		}

		msg := model.ThingActivity{}
		msg.Create(
			primitive.NilObjectID,
			note["data"].(map[string]interface{})["postID"].(primitive.ObjectID),
			ctx.Profile,
			strings.ToUpper(consts.NoteType),
			fmt.Sprintf("%s %s updated the note <b>%s</b>", cp.FirstName, cp.LastName, util.GetTitle(note["data"].(map[string]interface{}))),
		)
		err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
		if err != nil {
			return errors.Wrap(err, "unable to push to SQS")
		}
	}

	json.NewEncoder(w).Encode(note)
	return err
}

// DeleteNote - delete the text for the boards if the current user is creator of it
func (a *api) DeleteNote(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID, noteID, postID := ctx.Vars["boardID"], ctx.Vars["noteID"], ctx.Vars["postID"]
	if boardID == "" || noteID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	note, err := a.noteService.DeleteNote(boardID, noteID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "unable to delete note")
	}
	if note["status"].(int) != 0 {
		// delete from recentlyAdded, if present
		// _, err = a.recentThingsService.DeleteFromBoardThingsRecent(profileID, noteID, boardID)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to delete from Board's recent things")
		// }

		// flag from bookmarks, if present
		_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, noteID, time.Now())
		if err != nil {
			fmt.Println(err.Error())
			return errors.Wrap(err, "unable to update flag in bookmark")
		}

		// remove from board things tags
		_, err = a.boardService.DeleteFromBoardThingsTags(boardID, noteID)
		if err != nil {
			return errors.Wrap(err, "unable to delete from board things tags")
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

		// add to search results
		// err = a.searchService.UpdateSearchResults(nil, "delete", noteID)
		// if err != nil {
		// 	return err
		// }

		in := &searchrpc.UpdateSearchResultRequest{
			Data:       nil,
			UpdateType: "delete",
			Args:       noteID,
		}
		_, err = a.repos.SearchGrpcServiceClient.UpdateSearchResult(context.TODO(), in)
		if err != nil {
			return err
		}

		msg := model.ThingActivity{}
		msg.Create(
			primitive.NilObjectID,
			postObjID,
			ctx.Profile,
			strings.ToUpper(consts.NoteType),
			fmt.Sprintf("%s %s deleted the note <b>%s</b>", cp.FirstName, cp.LastName, util.GetTitle(note["data"].(map[string]interface{}))),
		)
		err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
		if err != nil {
			return errors.Wrap(err, "unable to push to SQS")
		}

		json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Note deleted successfully"))
		return nil
	}

	note["data"] = nil
	json.NewEncoder(w).Encode(note)

	return err
}

func (a *api) GetNoteByID(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	profileID := ctx.Profile
	if profileID == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	noteID := ctx.Vars["noteID"]
	if boardID == "" || noteID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	note, err := a.noteService.GetNoteByID(noteID, ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(note)
		return nil
	}

	return err
}
