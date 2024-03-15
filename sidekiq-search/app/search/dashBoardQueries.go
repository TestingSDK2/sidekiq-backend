package search

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 'plugging' values in the mongo search query
func GlobalSearchFTS(query string, profileIdStr string, filter *model.GlobalSearchFilter,
	collection *mongo.Collection, ctx context.Context, opts *options.AggregateOptions,
	allRetChan chan []map[string]interface{}, wg *sync.WaitGroup, offset, limitInt int, sortBy string, orderBy int) {
	defer wg.Done()
	p := bson.A{
		bson.M{
			"$search": bson.M{
				"index": searchIndex,
				"compound": bson.M{
					"minimumShouldMatch": 1,
				},
			},
		},
	}

	// get title, description in bson format
	basicBsons := getBsonObjectsForSearch(query, "title", "description", "fileName")
	if p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] == nil {
		p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] = make(bson.A, 0)
		p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] = basicBsons
	}

	// sortby
	sortFilter := make(bson.M)
	if sortBy != "" {
		sortFilter["$sort"] = bson.M{sortBy: orderBy}
	} else {
		sortFilter["$addFields"] = bson.M{"score": bson.M{"$meta": "searchScore"}}
	}
	p = append(p, sortFilter)

	if filter != nil {
		if p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"] == nil {
			p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"] = make(bson.A, 0)
		}

		// handling 'tags' filter
		if filter.Tags {
			tagsBson := getBsonObjectsForSearch(query, "tags")[0].(primitive.M)
			p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] =
				append(p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"].(bson.A),
					tagsBson)
		}

		// handing 'types'
		if len(filter.Types) > 0 {
			typesBson := getBsonObjectsForSearch(filter.Types, "type")[0].(primitive.M)
			p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"] =
				append(p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"].(bson.A),
					typesBson)

			// add body_raw if NOTE is true
			if util.Contains(filter.Types, "NOTE") {
				bodyBson := getBsonObjectsForSearch(query, "body_raw")[0].(primitive.M)
				p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] =
					append(p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"].(bson.A),
						bodyBson)

			}
		}

		// handing the shared, followed, owned boards filter
		if filter.BoardIDS != "" {
			idsBson := getBsonObjectsForSearch(filter.BoardIDS, "boardID")[0].(primitive.M)
			p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] =
				append(p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"].(bson.A),
					idsBson)
		}

		// handling the 'video' & 'image' filter
		if strings.Contains(filter.RegxPattern, "image") || strings.Contains(filter.RegxPattern, "video") {
			regBson := getBsonObjectsForSearch(filter.RegxPattern, "fileMime")[0].(primitive.M)
			p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"] =
				append(p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"].(bson.A),
					regBson)
		}

		// handing the date range
		dtRange := bson.M{
			"range": bson.M{
				"path": "createDate",
			},
		}
		if !filter.ISOStDt.IsZero() {
			dtRange["range"].(bson.M)["gte"] = filter.ISOStDt
		}
		if !filter.ISOEndDt.IsZero() {
			dtRange["range"].(bson.M)["lte"] = filter.ISOEndDt
		}

		p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"] =
			append(p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"].(bson.A),
				dtRange)

	}

	util.PrettyPrint(135, "final query", p)
	fmt.Println(strings.Repeat("*", 100))

	cursor, err := collection.Aggregate(ctx, p, opts)
	if err != nil {
		log.Fatal(err)
	}

	var results []map[string]interface{}
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}

	allRetChan <- results
}

func getBsonObjectsForSearch(query interface{}, paths ...string) bson.A {
	var bsons bson.A
	for _, p := range paths {
		bsons = append(bsons, bson.M{
			"text": bson.M{
				"query": query,
				"path":  p,
			},
		})
	}

	return bsons
}

func SearchConnection(query string, profiles []interface{}, collection *mongo.Collection,
	ctx context.Context, opts *options.AggregateOptions) []map[string]interface{} {
	p := bson.A{
		bson.M{
			"$search": bson.M{
				"index": connectionIdx,
				"compound": bson.M{
					"minimumShouldMatch": 1,
				},
			},
		},
	}

	bsons := getBsonObjectsForSearch(query, "birthday", "city",
		"email1", "firstName", "lastName", "metNotes",
		"nickName", "notes", "phone1", "relationship",
		"screenName", "state", "tags", "zip")

	if p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] == nil {
		p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] = make(bson.A, 0)

		p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"] =
			append(p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["should"].(bson.A),
				bsons...)
	}

	if p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"] == nil {
		p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"] = make(bson.A, 0)
		profilesBsons := getBsonObjectsForSearch(profiles, "profileID")[0].(primitive.M)
		p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"] =
			append(p[0].(bson.M)["$search"].(bson.M)["compound"].(bson.M)["must"].(bson.A),
				profilesBsons)
	}

	cursor, err := collection.Aggregate(ctx, p, opts)
	if err != nil {
		log.Fatal(err)
	}

	var results []map[string]interface{}
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}

	return results
}
