package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var sessionStore = make(map[string]*Session)

type Session struct {
	UserID         int
	OrganizationID int
	Role           string
	ExpiresAt      time.Time
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func CreateSession(userID, orgID int, role string) (string, error) {
	sessionID, err := GenerateSessionID()
	if err != nil {
		return "", err
	}

	session := &Session{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           role,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	sessionStore[sessionID] = session
	return sessionID, nil
}

func GetSession(sessionID string) (*Session, error) {
	session, exists := sessionStore[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		delete(sessionStore, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func DeleteSession(sessionID string) {
	delete(sessionStore, sessionID)
}

func GetSessionSecret() string {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		return "default-secret-change-in-production"
	}
	return secret
}
