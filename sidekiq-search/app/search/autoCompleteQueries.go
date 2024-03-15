package search

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetAutoCompleteResults(query string, collection *mongo.Collection,
	ctx context.Context, opts *options.AggregateOptions, allResultsChan chan []map[string]interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("query: ", query)
	searchPipeline := bson.A{
		bson.M{
			"$search": bson.M{
				"index": searchIndex,
				"compound": bson.M{
					"should": bson.A{
						bson.M{
							"autocomplete": bson.M{
								"query": query,
								"path":  "tags",
								"fuzzy": bson.M{"maxEdits": 1},
							},
						},
						bson.M{
							"autocomplete": bson.M{
								"query": query,
								"path":  "description",
								"fuzzy": bson.M{"maxEdits": 1},
							},
						},
						bson.M{
							"autocomplete": bson.M{
								"query": query,
								"path":  "title",
								"fuzzy": bson.M{"maxEdits": 1},
							},
						},
					},
				},
			},
		},
		bson.M{
			"$addFields": bson.M{
				"score": bson.M{"$meta": "searchScore"},
			},
		},
		// bson.M{
		// 	"$sample": bson.M{"size": 10},
		// },
		bson.M{
			"$sort": bson.M{"score": -1},
		},
		bson.M{"$limit": int64(10)},
	}

	cursor, err := collection.Aggregate(ctx, searchPipeline, opts)
	if err != nil {
		log.Fatal(err)
	}

	var results []map[string]interface{}
	if err = cursor.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}
	allResultsChan <- results
}
