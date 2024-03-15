package user

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
)

func getContacts(db *database.Database, userID int) ([]*model.Contact, error) {
	stmt := "SELECT id, userID, firstName, lastName, address, city, state, zip, country, phone, fax, email, lastModifiedDate FROM `sidekiq-dev`.Contact WHERE ownerID = ?;"
	contacts := []*model.Contact{}
	err := db.Conn.Select(&contacts, stmt, userID)
	return contacts, err
}
