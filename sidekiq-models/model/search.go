package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-search/consts"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/util"

	"github.com/pkg/errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getSearchDate(date string) (time.Time, error) {
	var err error
	dTcomps := strings.Split(date, "-")
	var year, mnth, day int
	if year, err = strconv.Atoi(dTcomps[0]); err != nil {
		return time.Time{}, errors.Wrap(err, "unable to convert string to int")
	}
	if mnth, err = strconv.Atoi(dTcomps[1]); err != nil {
		return time.Time{}, errors.Wrap(err, "unable to convert string to int")
	}
	if day, err = strconv.Atoi(dTcomps[2]); err != nil {
		return time.Time{}, errors.Wrap(err, "unable to convert string to int")
	}
	stDt := time.Date(year, time.Month(mnth), day, 0, 0, 0, 0, time.UTC)

	return stDt, nil
}

type SearchResult struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	BoardID     string             `json:"boardID" bson:"boardID"`
	Type        string             `json:"type" bson:"type"`
	Visible     string             `json:"visible" bson:"visible"`
	Tags        []string           `json:"tags" bson:"tags"`
	Description string             `json:"description" bson:"description"`
	Title       string             `json:"title" bson:"title"`
	Body_raw    string             `json:"body_raw" bson:"body_raw"`
}

type GlobalSearchFilter struct {
	Profiles       []interface{}          `json:"profiles"`
	Connections    bool                   `json:"connections"`
	Things         map[string]interface{} `json:"things"`
	Tags           bool                   `json:"tags"`
	MyBoards       bool                   `json:"myBoards"`
	SharedBoards   bool                   `json:"sharedBoards"`
	FollowedBoards bool                   `json:"followedBoards"`
	PublicBoards   bool                   `json:"publicBoards"`
	StartDate      string                 `json:"startDate"`
	EndDate        string                 `json:"endDate"`
	ISOStDt        time.Time
	ISOEndDt       time.Time
	Types          []string
	BoardIDS       string
	RegxPattern    string
}

type Bids []map[string]interface{}

func (b Bids) GetBoardsIdsHex(boards *[]string) {
	for _, value := range b {
		*boards = append(*boards, value["_id"].(primitive.ObjectID).Hex())
	}
}

// preparing the values for the search query
func (filter *GlobalSearchFilter) Process(db *mongodatabase.DBConfig, mysql *database.Database, wg *sync.WaitGroup, profileID int) error {
	var goroutines int
	var boards []string
	errChan := make(chan error)
	var err error
	profileIDStr := strconv.Itoa(profileID)
	regVals := ""

	filter.Profiles = append(filter.Profiles, profileIDStr)
	filter.RegxPattern = "^(%s)/"

	for k, v := range filter.Things {
		if k != "photo" && k != "video" {
			if v.(bool) {
				filter.Types = append(filter.Types, strings.ToUpper(k))
			}
		} else if v.(bool) {
			if !util.Contains(filter.Types, consts.FileType) {
				filter.Types = append(filter.Types, consts.FileType)
			}
			if k == "photo" {
				regVals += "image" + "|"
			} else {
				regVals += k + "|"
			}
		}
	}
	regVals = strings.TrimRight(regVals, "|")
	filter.RegxPattern = fmt.Sprintf(filter.RegxPattern, regVals)

	if filter.SharedBoards || filter.MyBoards || filter.PublicBoards {
		boardConn, err := db.New(consts.Board)
		if err != nil {
			return err
		}
		bc, boardClient := boardConn.Collection, boardConn.Client
		defer boardClient.Disconnect(context.TODO())

		var bids Bids
		opts := options.Find().SetProjection(
			bson.M{"_id": 1},
		)

		if filter.SharedBoards {
			// get shared boards from mongo
			wg.Add(1)
			goroutines += 1
			go func(errChan chan<- error) {
				f := bson.M{
					"$or": bson.A{
						bson.M{"viewers": bson.M{"$in": bson.A{filter.Profiles}}},
						bson.M{"subscribers": bson.M{"$in": bson.A{filter.Profiles}}},
						bson.M{"admins": bson.M{"$in": bson.A{filter.Profiles}}},
						bson.M{"guests": bson.M{"$in": bson.A{filter.Profiles}}},
					},
				}

				cur, err := bc.Find(context.TODO(), f, opts)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to find shared boards")
				}

				err = cur.All(context.TODO(), &bids)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to decode cursor")
				}
				defer cur.Close(context.TODO())

				bids.GetBoardsIdsHex(&boards)
				errChan <- nil
			}(errChan)
		}

		if filter.MyBoards {
			wg.Add(1)
			goroutines += 1
			go func(errChan chan<- error) {
				cur, err := bc.Find(context.TODO(), bson.M{"owner": bson.M{"$in": filter.Profiles}}, opts)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to find shared boards of profile")
				}
				err = cur.All(context.TODO(), &bids)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to decode cursor")
				}
				defer cur.Close(context.TODO())

				bids.GetBoardsIdsHex(&boards)
				errChan <- nil
			}(errChan)
		}

		if filter.PublicBoards {
			wg.Add(1)
			goroutines += 1
			go func(errChan chan<- error) {
				// get public board Ids
				cur, err := bc.Find(context.TODO(), bson.M{"state": consts.Public}, opts)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to decode cursor")
				}
				err = cur.All(context.TODO(), &bids)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to decode cursor")
				}
				defer cur.Close(context.TODO())

				bids.GetBoardsIdsHex(&boards)
				errChan <- nil
			}(errChan)
		}
	}

	if filter.FollowedBoards {
		wg.Add(1)
		goroutines += 1
		go func(errChan chan<- error) {
			// get followed boards from mysql
			stmt := fmt.Sprintf("SELECT boardID FROM `sidekiq-dev`.BoardsFollowed WHERE profileID IN (%s)",
				strings.TrimRight(strings.Repeat(" ?,", len(filter.Profiles)), ", "))

			fmt.Println(161)
			fmt.Println(stmt)
			fmt.Println(filter.Profiles...)

			err = mysql.Conn.Select(&boards, stmt, filter.Profiles...)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch followed boards")
			}
			errChan <- nil
		}(errChan)
	}

	// waiting for goroutines to finish
	for i := 0; i < goroutines; i++ {
		if err := <-errChan; err != nil {
			return errors.Wrap(err, "error from processing search filter")
		}
	}

	for _, dat := range boards {
		filter.BoardIDS += dat + " OR "
	}
	filter.BoardIDS = strings.TrimRight(filter.BoardIDS, " OR ")

	// parse 'start date' & 'end date' dates
	if filter.StartDate != "" {
		filter.ISOStDt, err = getSearchDate(filter.StartDate)
		if err != nil {
			return errors.Wrap(err, "unable to parse date as per Global search")
		}
	} else {
		filter.ISOStDt = time.Time{}
	}
	if filter.EndDate != "" {
		filter.ISOEndDt, err = getSearchDate(filter.EndDate)
		if err != nil {
			return errors.Wrap(err, "unable to parse date as per Global search")
		}

		// end date should be always be larger than start date
		if filter.ISOEndDt.Sub(filter.ISOStDt).Milliseconds() < 0 {
			return errors.New("end date cannot be lower than start date")
		}
	} else {
		filter.ISOEndDt = time.Time{}
	}

	return nil
}
