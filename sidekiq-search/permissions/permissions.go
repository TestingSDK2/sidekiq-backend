package permissions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/util"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// caches the permissions of the newly created board members along with its parent board members, if any.
func CacheBoardsPermissions(cache *cache.Cache, cacheParents bool, parentBoardIds []map[string]interface{}, boardIdObj primitive.ObjectID, profileID int, board model.Board, removedMembers []string) error {
	for i := 0; i < len(removedMembers); i++ {
		BoardPermission := make(model.BoardPermission)
		Key := fmt.Sprintf("boards:%s", removedMembers[i])
		CachedBoards, _ := cache.GetValue(Key)
		if CachedBoards != "" {
			_ = json.Unmarshal([]byte(CachedBoards), &BoardPermission)
			delete(BoardPermission, boardIdObj.Hex())
			cache.SetValue(Key, BoardPermission.ToJSON())
		}
	}
	// **************** CACHES PERMISSIONS OF PARENTS OF THE NEWLY CREATED BOARD **********************
	if cacheParents {
		for _, parentBoardId := range parentBoardIds {
			for _, pb := range parentBoardId["parentBoards"].(primitive.A) {
				pbObj := pb.(map[string]interface{})
				// propagating parent board's permission into the child board
				ownerBoardPermission := make(model.BoardPermission)
				ownerKey := fmt.Sprintf("boards:%s", pbObj["owner"].(string))
				ownerCachedBoards, _ := cache.GetValue(ownerKey)
				if ownerCachedBoards != "" { // profile is logged in
					_ = json.Unmarshal([]byte(ownerCachedBoards), &ownerBoardPermission)
					ownerBoardPermission[boardIdObj.Hex()] = "admin"
					cache.SetValue(ownerKey, ownerBoardPermission.ToJSON())
				}

				// checking for parent authors
				authorBoardPermission := make(model.BoardPermission)
				if pbObj["authors"] != nil {
					for _, author := range pbObj["authors"].(primitive.A) {
						authorKey := fmt.Sprintf("boards:%s", author.(string))
						authorCachedBoards, _ := cache.GetValue(authorKey)
						if authorCachedBoards != "" {
							_ = json.Unmarshal([]byte(authorCachedBoards), &authorBoardPermission)
							authorBoardPermission[boardIdObj.Hex()] = "author"
							cache.SetValue(authorKey, authorBoardPermission.ToJSON())
						}
					}
				}

				// checking for parent viewers
				viewerBoardPermission := make(model.BoardPermission)
				if pbObj["viewers"] != nil {
					for _, viewer := range pbObj["viewers"].(primitive.A) {
						viewerKey := fmt.Sprintf("boards:%s", viewer.(string))
						viewerCachedBoards, _ := cache.GetValue(viewerKey)
						if viewerCachedBoards != "" {
							_ = json.Unmarshal([]byte(viewerCachedBoards), &viewerBoardPermission)
							viewerBoardPermission[boardIdObj.Hex()] = "viewer"
							cache.SetValue(viewerKey, viewerBoardPermission.ToJSON())
						}
					}
				}

				// checking for parent admins
				adminBoardPermission := make(model.BoardPermission)
				if pbObj["admins"] != nil {
					for _, admin := range pbObj["admins"].(primitive.A) {
						adminKey := fmt.Sprintf("boards:%s", admin.(string))
						adminCachedBoards, _ := cache.GetValue(adminKey)
						if adminCachedBoards != "" {
							_ = json.Unmarshal([]byte(adminCachedBoards), &adminBoardPermission)
							adminBoardPermission[boardIdObj.Hex()] = "admin"
							cache.SetValue(adminKey, adminBoardPermission.ToJSON())
						}
					}
				}

				// checking for parent subscribers
				subsBoardPermission := make(model.BoardPermission)
				if pbObj["subscribers"] != nil {
					for _, sub := range pbObj["subscribers"].(primitive.A) {
						subKey := fmt.Sprintf("boards:%s", sub.(string))
						subsCachedBoards, _ := cache.GetValue(subKey)
						if subsCachedBoards != "" {
							_ = json.Unmarshal([]byte(subsCachedBoards), &subsBoardPermission)
							subsBoardPermission[boardIdObj.Hex()] = "subscriber"
							cache.SetValue(subKey, subsBoardPermission.ToJSON())
						}
					}
				}
			}
		} // end of parent boards for loop
	}

	// **************** CACHES PERMISSIONS OF THE NEWLY CREATED BOARD **********************
	boardId := boardIdObj.Hex()
	key := fmt.Sprintf("boards:%s", board.Owner)
	val, _ := cache.GetValue(key) // taking existing board permissions
	loggedInProfileBoardPermission := make(model.BoardPermission)
	_ = json.Unmarshal([]byte(val), &loggedInProfileBoardPermission)
	loggedInProfileBoardPermission[boardId] = "owner"
	cache.SetValue(key, loggedInProfileBoardPermission.ToJSON())
	fmt.Println(loggedInProfileBoardPermission.ToJSON())

	// caching other members
	authorBoardPermission := make(model.BoardPermission)
	if len(board.Authors) != 0 {
		for _, author := range board.Authors {
			authorKey := fmt.Sprintf("boards:%s", author)
			authorCachedBoards, _ := cache.GetValue(authorKey)
			if authorCachedBoards != "" {
				_ = json.Unmarshal([]byte(authorCachedBoards), &authorBoardPermission)
				authorBoardPermission[boardId] = "author"
				cache.SetValue(authorKey, authorBoardPermission.ToJSON())
			}
		}
	}

	viewerBoardPermission := make(model.BoardPermission)
	if len(board.Viewers) != 0 {
		for _, viewer := range board.Viewers {
			viewerKey := fmt.Sprintf("boards:%s", viewer)
			viewerCachedBoards, _ := cache.GetValue(viewerKey)
			if viewerCachedBoards != "" {
				_ = json.Unmarshal([]byte(viewerCachedBoards), &viewerBoardPermission)
				viewerBoardPermission[boardId] = "viewer"
				setErr := cache.SetValue(viewerKey, viewerBoardPermission.ToJSON())
				if setErr != nil {
					return setErr
				}
			}
		}
	}

	adminBoardPermission := make(model.BoardPermission)
	if len(board.Admins) != 0 {
		for _, admin := range board.Admins {
			adminKey := fmt.Sprintf("boards:%s", admin)
			adminCachedBoards, _ := cache.GetValue(adminKey)
			if adminCachedBoards != "" {
				_ = json.Unmarshal([]byte(adminCachedBoards), &adminBoardPermission)
				adminBoardPermission[boardId] = "admin"
				cache.SetValue(adminKey, adminBoardPermission.ToJSON())
			}
		}
	}

	subsBoardPermission := make(model.BoardPermission)
	if len(board.Subscribers) != 0 {
		for _, sub := range board.Subscribers {
			subKey := fmt.Sprintf("boards:%s", sub)
			subsCachedBoards, _ := cache.GetValue(subKey)
			if subsCachedBoards != "" {
				_ = json.Unmarshal([]byte(subsCachedBoards), &subsBoardPermission)
				subsBoardPermission[boardId] = "subscriber"
				cache.SetValue(subKey, subsBoardPermission.ToJSON())
			}
		}
	}
	return nil
}

// returns board permissions from redis in map[string]interface{} format
func GetBoardPermissions(key string, cache *cache.Cache) model.BoardPermission {
	val, _ := cache.GetValue(key)
	boardPermissions := make(model.BoardPermission)
	if val != "" {
		jsonErr := json.Unmarshal([]byte(val), &boardPermissions)
		if jsonErr != nil {
			return nil
		}
	} else {
		// get krke cache krna h
	}
	return boardPermissions
}

// returns board permissions from redis in map[string]interface{} format and if not found in redis then fetch from Mongo
func GetBoardPermissionsNew(key string, cache *cache.Cache, boardObj *model.Board, profileID string) model.BoardPermission {
	val, _ := cache.GetValue(key)
	boardPermissions := make(model.BoardPermission)
	if val != "" {
		jsonErr := json.Unmarshal([]byte(val), &boardPermissions)
		if jsonErr != nil {
			return nil
		}
	}
	if boardPermissions[boardObj.Id.Hex()] == "" {
		//owner, admin, author, subscriber, guest, /* viewer */, blocked, followers
		role := ""
		if profileID == boardObj.Owner {
			role = "owner"
		} else if util.Contains(boardObj.Admins, profileID) {
			role = "admin"
		} else if util.Contains(boardObj.Authors, profileID) {
			role = "author"
		} else if util.Contains(boardObj.Subscribers, profileID) {
			role = "subscriber"
		} else if util.Contains(boardObj.Guests, profileID) || util.Contains(boardObj.Followers, profileID) {
			role = "viewer"
		} else if util.Contains(boardObj.Blocked, profileID) {
			role = "blocked"
		}
		if role == "" {
			return boardPermissions
		}
		// set cache
		boardPermissions[boardObj.Id.Hex()] = role
		err := cache.SetValue(key, boardPermissions.ToJSON())
		if err != nil {
			fmt.Println("error in setting cache value", err)
			return nil
		}
	}
	return boardPermissions
}

// delete board permissions for all of the members of that board on deletion of that board
func DeleteBoardPermissions(cache *cache.Cache, cacheParents bool, parentBoardIds []map[string]interface{}, boardIdObj primitive.ObjectID, profileID int, board model.Board) error {
	if cacheParents {
		for _, parentBoardId := range parentBoardIds {
			for _, pb := range parentBoardId["parentBoards"].(primitive.A) {
				pbObj := pb.(map[string]interface{})
				// propagating parent board's permission into the child board
				ownerBoardPermission := make(model.BoardPermission)
				ownerKey := fmt.Sprintf("boards:%s", pbObj["owner"].(string))
				ownerCachedBoards, _ := cache.GetValue(ownerKey)
				if ownerCachedBoards != "" { // profile is logged in
					_ = json.Unmarshal([]byte(ownerCachedBoards), &ownerBoardPermission)
					delete(ownerBoardPermission, boardIdObj.Hex())
					cache.SetValue(ownerKey, ownerBoardPermission.ToJSON())
				}

				// checking for parent authors
				authorBoardPermission := make(model.BoardPermission)
				if pbObj["authors"] != nil {
					for _, author := range pbObj["authors"].(primitive.A) {
						authorKey := fmt.Sprintf("boards:%s", author.(string))
						authorCachedBoards, _ := cache.GetValue(authorKey)
						if authorCachedBoards != "" {
							_ = json.Unmarshal([]byte(authorCachedBoards), &authorBoardPermission)
							delete(authorBoardPermission, boardIdObj.Hex())
							cache.SetValue(authorKey, authorBoardPermission.ToJSON())
						}
					}
				}

				// checking for parent viewers
				viewerBoardPermission := make(model.BoardPermission)
				if pbObj["viewers"] != nil {
					for _, viewer := range pbObj["viewers"].(primitive.A) {
						viewerKey := fmt.Sprintf("boards:%s", viewer.(string))
						viewerCachedBoards, _ := cache.GetValue(viewerKey)
						if viewerCachedBoards != "" {
							_ = json.Unmarshal([]byte(viewerCachedBoards), &viewerBoardPermission)
							delete(viewerBoardPermission, boardIdObj.Hex())
							cache.SetValue(viewerKey, viewerBoardPermission.ToJSON())
						}
					}
				}

				// checking for parent admins
				adminBoardPermission := make(model.BoardPermission)
				if pbObj["admins"] != nil {
					for _, admin := range pbObj["admins"].(primitive.A) {
						adminKey := fmt.Sprintf("boards:%s", admin.(string))
						adminCachedBoards, _ := cache.GetValue(adminKey)
						if adminCachedBoards != "" {
							_ = json.Unmarshal([]byte(adminCachedBoards), &adminBoardPermission)
							delete(adminBoardPermission, boardIdObj.Hex())
							cache.SetValue(adminKey, adminBoardPermission.ToJSON())
						}
					}
				}

				// checking for parent subscribers
				subsBoardPermission := make(model.BoardPermission)
				if pbObj["subscribers"] != nil {
					for _, sub := range pbObj["subscribers"].(primitive.A) {
						subKey := fmt.Sprintf("boards:%s", sub.(string))
						subsCachedBoards, _ := cache.GetValue(subKey)
						if subsCachedBoards != "" {
							_ = json.Unmarshal([]byte(subsCachedBoards), &subsBoardPermission)
							delete(subsBoardPermission, boardIdObj.Hex())
							cache.SetValue(subKey, subsBoardPermission.ToJSON())
						}
					}
				}
			}
		} // end of parent boards for loop
	}

	key := fmt.Sprintf("boards:%s", board.Owner)
	val, _ := cache.GetValue(key)
	loggedInProfileBoardPermission := make(model.BoardPermission)
	_ = json.Unmarshal([]byte(val), &loggedInProfileBoardPermission)
	delete(loggedInProfileBoardPermission, boardIdObj.Hex())
	cache.SetValue(key, loggedInProfileBoardPermission.ToJSON())

	authorBoardPermission := make(model.BoardPermission)
	if len(board.Authors) != 0 {
		for _, author := range board.Authors {
			authorKey := fmt.Sprintf("boards:%s", author)
			authorCachedBoards, _ := cache.GetValue(authorKey)
			if authorCachedBoards != "" {
				_ = json.Unmarshal([]byte(authorCachedBoards), &authorBoardPermission)
				delete(authorBoardPermission, boardIdObj.Hex())
				cache.SetValue(authorKey, authorBoardPermission.ToJSON())
			}
		}
	}

	viewerBoardPermission := make(model.BoardPermission)
	if len(board.Viewers) != 0 {
		for _, viewer := range board.Viewers {
			viewerKey := fmt.Sprintf("boards:%s", viewer)
			viewerCachedBoards, _ := cache.GetValue(viewerKey)
			if viewerCachedBoards != "" {
				_ = json.Unmarshal([]byte(viewerCachedBoards), &viewerBoardPermission)
				delete(viewerBoardPermission, boardIdObj.Hex())
				setErr := cache.SetValue(viewerKey, viewerBoardPermission.ToJSON())
				if setErr != nil {
					return setErr
				}
			}
		}
	}

	adminBoardPermission := make(model.BoardPermission)
	if len(board.Admins) != 0 {
		for _, admin := range board.Admins {
			adminKey := fmt.Sprintf("boards:%s", admin)
			adminCachedBoards, _ := cache.GetValue(adminKey)
			if adminCachedBoards != "" {
				_ = json.Unmarshal([]byte(adminCachedBoards), &adminBoardPermission)
				delete(adminBoardPermission, boardIdObj.Hex())
				cache.SetValue(adminKey, adminBoardPermission.ToJSON())
			}
		}
	}

	subsBoardPermission := make(model.BoardPermission)
	if len(board.Subscribers) != 0 {
		for _, sub := range board.Subscribers {
			subKey := fmt.Sprintf("boards:%s", sub)
			subsCachedBoards, _ := cache.GetValue(subKey)
			if subsCachedBoards != "" {
				_ = json.Unmarshal([]byte(subsCachedBoards), &subsBoardPermission)
				delete(subsBoardPermission, boardIdObj.Hex())
				cache.SetValue(subKey, subsBoardPermission.ToJSON())
			}
		}
	}
	return nil
}

func CheckValidPermissions(profileID string, cache *cache.Cache, coll *mongo.Collection, Id string, roles []string, isViewOnly bool) (bool, error) {
	var hasValidPermissions bool
	key := fmt.Sprintf("boards:%s", profileID)
	var board *model.Board
	boardObjID, err := primitive.ObjectIDFromHex(Id)
	if err != nil {
		return false, errors.Wrap(err, "unable to convert string to ObjectID")
	}
	fmt.Println("boardID", Id)
	err = coll.FindOne(context.TODO(), bson.M{"_id": boardObjID}).Decode(&board)
	if err != nil {
		return false, errors.Wrap(err, "unable to find board")
	}

	boardPermissions := GetBoardPermissionsNew(key, cache, board, profileID)
	role := boardPermissions[Id]
	fmt.Println("ROLE -> ", role)
	if role == "" {
		hasValidPermissions = true
		return hasValidPermissions, nil
	}

	// only for viewing things
	if isViewOnly {
		if role != "blocked" {
			hasValidPermissions = true
			return hasValidPermissions, nil
		}
	} else {
		for _, r := range roles {
			if role == r { // if any one permission if valid, then it's good
				hasValidPermissions = true
				return hasValidPermissions, nil
			}
		}
	}

	hasValidPermissions = false
	return hasValidPermissions, nil
}
