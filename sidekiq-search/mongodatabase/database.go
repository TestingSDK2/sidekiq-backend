package mongodatabase

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" // Load the mysql driver
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBConn struct {
	Collection *mongo.Collection `mapstructure:"collection"`
	Client     *mongo.Client     `mapstructure:"client"`
}

// New create new DB
func (config *DBConfig) New(collectionName string) (dbconn *MongoDBConn, err error) {
	// clientOptions := options.Client().ApplyURI(config.Host).SetConnectTimeout(10 * time.Minute).SetSocketTimeout(10 * time.Minute)
	clientOptions := options.Client().ApplyURI(config.Host).
		SetRetryReads(true).
		SetRetryWrites(true).
		SetConnectTimeout(5 * time.Minute)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)

	if err != nil {
		log.Fatal(err)
	}

	// Check the connection; Date:   Mon Mar 15 14:29:53 2021 +0530
	err = client.Ping(context.TODO(), nil)

	if err != nil {
		log.Fatal(err, "error connecting mongo")
		return &MongoDBConn{}, err
	}

	collection := client.Database(config.DBName).Collection(collectionName)
	fmt.Printf("Connected to %s\n", collection.Name())

	return &MongoDBConn{collection, client}, nil
}

// Close DB
func Close(c *mongo.Client) error {
	return c.Disconnect(context.TODO())
}

// func getDatabaseDSN(config *DBConfig) string {
// 	return fmt.Sprintf("%s:%s@tcp(%s:%s)/", config.UserName, config.Password, config.Host, config.Port)
// }

// // QuerySingle - run query and return single row
// func (d *Database) QuerySingle(stmt string, args ...interface{}) *sql.Row {
// 	row := d.Conn.QueryRow(stmt, args...)
// 	return row
// }

// // Query - run query and return multiple rows
// func (d *Database) Query(stmt string, args ...interface{}) *sql.Rows {
// 	rows, err := d.Conn.Query(stmt, args...)
// 	if err != nil {
// 		return nil
// 	}
// 	return rows
// }
