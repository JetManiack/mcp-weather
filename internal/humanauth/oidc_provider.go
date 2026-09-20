package humanauth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/storage"
)

// SessionCookieName is the name of the cookie carrying the session ID.
const SessionCookieName = "weather_session"

// sessionRefreshInterval is how long a session's cached claims are trusted
// before Authenticate re-validates them against Keycloak via the stored
// refresh token.
const sessionRefreshInterval = 15 * time.Minute

type tokenRefresher interface {
	Refresh(refreshToken string) (identity Identity, newRefreshToken string, err error)
}

// OIDCProvider authenticates requests via a server-side session cookie,
// re-validating the session's claims against Keycloak once
// sessionRefreshInterval has elapsed.
type OIDCProvider struct {
	db            *gorm.DB
	refresher     tokenRefresher
	encryptionKey []byte
}

// NewOIDCProvider constructs an OIDCProvider.
func NewOIDCProvider(db *gorm.DB, refresher tokenRefresher, encryptionKey []byte) OIDCProvider {
	return OIDCProvider{db: db, refresher: refresher, encryptionKey: encryptionKey}
}

func hashSessionID(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// HashSessionIDForTesting exposes hashSessionID for tests.
func HashSessionIDForTesting(raw string) string {
	return hashSessionID(raw)
}

func (p OIDCProvider) Authenticate(r *http.Request) (*Identity, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, errors.New("no session cookie")
	}

	var session storage.Session
	if err := p.db.First(&session, "id = ?", hashSessionID(cookie.Value)).Error; err != nil {
		return nil, errors.New("no such session")
	}

	if time.Now().Before(session.ExpiresAt) {
		return &Identity{
			Subject:     session.Subject,
			DisplayName: session.DisplayName,
			Role:        session.Role,
		}, nil
	}

	refreshToken, err := decryptRefreshToken(p.encryptionKey, session.RefreshToken)
	if err != nil {
		p.db.Delete(&session)
		return nil, fmt.Errorf("decrypt stored refresh token: %w", err)
	}

	identity, newRefreshToken, err := p.refresher.Refresh(refreshToken)
	if err != nil {
		p.db.Delete(&session)
		return nil, errors.New("session refresh failed: " + err.Error())
	}

	encryptedNewToken, err := encryptRefreshToken(p.encryptionKey, newRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("encrypt refreshed token: %w", err)
	}

	session.Subject = identity.Subject
	session.DisplayName = identity.DisplayName
	session.Role = identity.Role
	session.RefreshToken = encryptedNewToken
	session.ExpiresAt = time.Now().Add(sessionRefreshInterval)
	if err := p.db.Save(&session).Error; err != nil {
		return nil, err
	}

	return &identity, nil
}
