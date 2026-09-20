package humanauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JetManiack/mcp-weather/internal/humanauth"
	"github.com/JetManiack/mcp-weather/internal/storage"
)

// stubProvider wraps StubProvider and records calls.
type countingProvider struct {
	humanauth.StubProvider
	calls int
}

func (c *countingProvider) Authenticate(r *http.Request) (*humanauth.Identity, error) {
	c.calls++
	return c.StubProvider.Authenticate(r)
}

func TestRequireHumanAuthPassesWithStub(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	var actorID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := humanauth.ActorFromContext(r.Context())
		if !ok {
			t.Error("no actor in context")
		} else {
			actorID = actor.ID
		}
		role, ok := humanauth.RoleFromContext(r.Context())
		if !ok || role != "admin" {
			t.Errorf("expected admin role, got %q (ok=%v)", role, ok)
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := humanauth.RequireHumanAuth(db, humanauth.StubProvider{})(inner)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if actorID == "" {
		t.Fatal("actor ID was not set")
	}
}

func TestRequireAdminForbidsViewer(t *testing.T) {
	viewerProvider := viewerStub{}
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	inner := humanauth.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not have reached admin handler")
		w.WriteHeader(http.StatusOK)
	}))
	handler := humanauth.RequireHumanAuth(db, viewerProvider)(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/actors", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

type viewerStub struct{}

func (viewerStub) Authenticate(r *http.Request) (*humanauth.Identity, error) {
	return &humanauth.Identity{
		Subject:     "viewer-subject",
		DisplayName: "Viewer",
		Role:        "viewer",
	}, nil
}
