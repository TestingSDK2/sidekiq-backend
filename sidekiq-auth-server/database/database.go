package database

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql" // Load the mysql driver
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Database mysql DB
type Database struct {
	Conn *sqlx.DB
}

// New create new DB
func New(config *DBConfig) (*Database, error) {
	// Initialise a new connection pool
	db, err := sqlx.Connect(config.Type, getDatabaseDSN(config))
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to database")
	}
	// We can set these config params as we need
	db.SetConnMaxLifetime(config.MaxLifetime)
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)

	return &Database{
		Conn: db,
	}, nil
}

// Close DB
func (d *Database) Close() error {
	return d.Conn.Close()
}

func getDatabaseDSN(config *DBConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", config.UserName, config.Password, config.Host, config.Port)
}

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
