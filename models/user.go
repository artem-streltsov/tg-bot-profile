package models

import "time"

type User struct {
	ID            int64
	FirstName     string
	LastName      string
	UserName      string
	ZodiacSign    string
	BirthDate     string
	BirthTime     string
	IsPremium     bool
	PremiumExpiry time.Time
}
