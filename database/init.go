package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

func InitTables(db *sql.DB) error {

	if _, err := createTables(db); err != nil {
		return err
	}
	return nil
}

func createTables(db *sql.DB) (string, error) {

	//Create Event Table
	if _, err := db.Query(CreateEventTable); err != nil {
		return "", err
	}

	//Create Teams Table
	if _, err := db.Query(CreateTeamsTable); err != nil {
		return "", err
	}

	//Create Profiles Table
	if _, err := db.Query(CreateProfilesTable); err != nil {
		return "", err
	}

	return OK, nil
}
