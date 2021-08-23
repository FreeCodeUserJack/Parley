package dto

import (
	"html"
	"strings"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Sanitize
func (l *LoginRequest) Sanitize() {
	l.Email = strings.TrimSpace(html.EscapeString(l.Email))
	// l.Password = strings.TrimSpace(html.EscapeString(l.Password))
}

// Validate
func (l LoginRequest) Validate() bool {
	if l.Email == "" || l.Password == "" {
		return false
	}

	return true
}
