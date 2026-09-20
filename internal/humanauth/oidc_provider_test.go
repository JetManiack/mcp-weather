package humanauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/JetManiack/mcp-weather/internal/humanauth"
	"github.com/JetManiack/mcp-weather/internal/storage"
)

type nopRefresher struct{}

func (nopRefresher) Refresh(refreshToken string) (humanauth.Identity, string, error) {
	return humanauth.Identity{
		Subject:     "sub",
		DisplayName: "Test User",
		Role:        "admin",
	}, refreshToken, nil
}

func TestOIDCProviderAuthenticateFromSession(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	key := make([]byte, 32)
	encryptedToken, err := humanauth.EncryptRefreshTokenForTesting(key, "rt-value")
	if err != nil {
		t.Fatal(err)
	}

	rawID := "test-session-id-raw"
	session := &storage.Session{
		ID:           humanauth.HashSessionIDForTesting(rawID),
		Subject:      "sub",
		DisplayName:  "Test User",
		Role:         "admin",
		RefreshToken: encryptedToken,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	if err := db.Create(session).Error; err != nil {
		t.Fatal(err)
	}

	provider := humanauth.NewOIDCProvider(db, nopRefresher{}, key)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  humanauth.SessionCookieName,
		Value: rawID,
	})

	identity, err := provider.Authenticate(req)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if identity.Subject != "sub" {
		t.Errorf("want subject=sub, got %q", identity.Subject)
	}
	if identity.Role != "admin" {
		t.Errorf("want role=admin, got %q", identity.Role)
	}
}

func TestOIDCProviderRejectsNoCookie(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	provider := humanauth.NewOIDCProvider(db, nopRefresher{}, make([]byte, 32))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err = provider.Authenticate(req)
	if err == nil {
		t.Fatal("expected error with no session cookie")
	}
}
