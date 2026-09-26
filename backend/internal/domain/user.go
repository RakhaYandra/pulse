package domain

import "time"

type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
}

func (u User) ValidateNew(password string) error {
	if u.Email == "" {
		return &FieldError{Field: "email", Message: "email invalid"}
	}
	if len(password) < 8 {
		return &FieldError{Field: "password", Message: "password min 8 chars"}
	}
	if u.Name == "" {
		return &FieldError{Field: "name", Message: "name required"}
	}
	return nil
}
