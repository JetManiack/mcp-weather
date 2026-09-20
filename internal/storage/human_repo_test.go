package storage_test

import (
	"testing"

	"github.com/JetManiack/mcp-weather/internal/storage"
)

func TestGetOrCreateHumanActorCreates(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	actor, err := storage.GetOrCreateHumanActor(db, "sub-1", "Alice", "admin")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if actor.Name != "Alice" {
		t.Errorf("want name=Alice, got %q", actor.Name)
	}
	if actor.Kind != storage.ActorKindHuman {
		t.Errorf("want kind=human, got %q", actor.Kind)
	}
}

func TestGetOrCreateHumanActorIdempotent(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	a1, err := storage.GetOrCreateHumanActor(db, "sub-2", "Bob", "viewer")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	a2, err := storage.GetOrCreateHumanActor(db, "sub-2", "Bob", "admin")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if a1.ID != a2.ID {
		t.Errorf("expected same actor ID on repeat call; got %q then %q", a1.ID, a2.ID)
	}

	// Role should have been updated to admin.
	var identity storage.UserIdentity
	if err := db.Where("keycloak_subject = ?", "sub-2").First(&identity).Error; err != nil {
		t.Fatal(err)
	}
	if identity.Role != "admin" {
		t.Errorf("expected role=admin after update, got %q", identity.Role)
	}
}

func TestGetOrCreateHumanActorDistinctSubjects(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	a1, _ := storage.GetOrCreateHumanActor(db, "sub-3", "Charlie", "viewer")
	a2, _ := storage.GetOrCreateHumanActor(db, "sub-4", "Diana", "admin")
	if a1.ID == a2.ID {
		t.Error("different subjects should produce different actors")
	}
}
