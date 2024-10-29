package database

import (
	"database/sql"
	"errors"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"tg-bot-profile/models"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	createTableQuery := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY,
        first_name TEXT,
        last_name TEXT,
        username TEXT,
        zodiac_sign TEXT,
        birth_date TEXT,
        birth_time TEXT,
        is_premium BOOLEAN,
        premium_expiry DATETIME
    );
    `

	_, err = DB.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
}

func SaveUser(user *models.User) error {
	query := `
    INSERT OR REPLACE INTO users (
        id, first_name, last_name, username, zodiac_sign, birth_date, birth_time, is_premium, premium_expiry
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
    `
	_, err := DB.Exec(query, user.ID, user.FirstName, user.LastName, user.UserName, user.ZodiacSign, user.BirthDate, user.BirthTime, user.IsPremium, user.PremiumExpiry)
	return err
}

func GetUser(userID int64) (*models.User, error) {
	query := `SELECT * FROM users WHERE id = ?;`
	row := DB.QueryRow(query, userID)

	var user models.User
	var premiumExpiry sql.NullTime
	err := row.Scan(&user.ID, &user.FirstName, &user.LastName, &user.UserName, &user.ZodiacSign, &user.BirthDate, &user.BirthTime, &user.IsPremium, &premiumExpiry)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if premiumExpiry.Valid {
		user.PremiumExpiry = premiumExpiry.Time
	}

	return &user, nil
}
