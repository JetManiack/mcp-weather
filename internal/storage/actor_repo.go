package storage

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidToken = errors.New("invalid or revoked token")
	ErrEmptyName    = errors.New("name must not be empty")
)

// TokenPrefix marks a raw bearer token as belonging to this service so a
// leaked token is recognizable in a log or a secret scanner.
const TokenPrefix = "wx_"

func CreateAgent(db *gorm.DB, name string) (*Actor, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrEmptyName
	}
	actor := &Actor{ID: uuid.NewString(), Name: name, Kind: ActorKindAgent}
	if err := db.Create(actor).Error; err != nil {
		return nil, err
	}
	return actor, nil
}

func ListAgents(db *gorm.DB) ([]Actor, error) {
	agents := []Actor{}
	if err := db.Where("kind = ?", ActorKindAgent).Order("created_at").Find(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}

func ListAgentCredentials(db *gorm.DB, actorID string) ([]AgentCredential, error) {
	creds := []AgentCredential{}
	if err := db.Where("actor_id = ?", actorID).Order("created_at DESC").Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

func ActorNamesByID(db *gorm.DB, ids []string) (map[string]string, error) {
	names := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	var actors []Actor
	if err := db.Where("id IN ?", ids).Find(&actors).Error; err != nil {
		return nil, err
	}
	for _, a := range actors {
		names[a.ID] = a.Name
	}
	return names, nil
}

func ActorsWithActiveToken(db *gorm.DB) (map[string]bool, error) {
	var ids []string
	if err := db.Model(&AgentCredential{}).
		Where("revoked_at IS NULL").
		Distinct("actor_id").
		Pluck("actor_id", &ids).Error; err != nil {
		return nil, err
	}
	active := make(map[string]bool, len(ids))
	for _, id := range ids {
		active[id] = true
	}
	return active, nil
}

func RevokeAgentToken(db *gorm.DB, actorID, credentialID string) error {
	return db.Model(&AgentCredential{}).
		Where("id = ? AND actor_id = ? AND revoked_at IS NULL", credentialID, actorID).
		Update("revoked_at", time.Now()).Error
}

// RevokeCredential revokes a credential by its own ID (no actor-ID check).
func RevokeCredential(db *gorm.DB, credentialID string) error {
	return db.Model(&AgentCredential{}).
		Where("id = ? AND revoked_at IS NULL", credentialID).
		Update("revoked_at", time.Now()).Error
}

func RevokeAllAgentCredentials(db *gorm.DB, actorID string) error {
	return db.Model(&AgentCredential{}).
		Where("actor_id = ? AND revoked_at IS NULL", actorID).
		Update("revoked_at", time.Now()).Error
}
