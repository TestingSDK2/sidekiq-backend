package user

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
)

func getContacts(db *database.Database, userID int) ([]*model.Contact, error) {
	stmt := "SELECT id, userID, firstName, lastName, address, city, state, zip, country, phone, fax, email, lastModifiedDate FROM `sidekiq-dev`.Contact WHERE ownerID = ?;"
	contacts := []*model.Contact{}
	err := db.Conn.Select(&contacts, stmt, userID)
	return contacts, err
}
