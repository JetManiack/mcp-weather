package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidToken     = errors.New("invalid or revoked token")
	ErrEmptyDisplayName = errors.New("display name must not be empty")
)

// TokenPrefix marks a raw bearer token as belonging to this service so a
// leaked token is recognizable in a log or a secret scanner.
const TokenPrefix = "wx_"

func CreateAgent(db *gorm.DB, displayName string) (*Actor, error) {
	if strings.TrimSpace(displayName) == "" {
		return nil, ErrEmptyDisplayName
	}
	actor := &Actor{ID: uuid.NewString(), DisplayName: displayName, Kind: ActorKindAgent}
	if err := db.Create(actor).Error; err != nil {
		return nil, err
	}
	return actor, nil
}

// IssueAgentToken generates a new bearer token for actorID and stores only
// its hash. The raw token is returned once and never stored.
func IssueAgentToken(db *gorm.DB, actorID string) (rawToken string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	rawToken = TokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	cred := &AgentCredential{
		ID:        uuid.NewString(),
		ActorID:   actorID,
		TokenHash: hashToken(rawToken),
	}
	if err := db.Create(cred).Error; err != nil {
		return "", err
	}
	return rawToken, nil
}

func AuthenticateAgentToken(db *gorm.DB, rawToken string) (*Actor, error) {
	var cred AgentCredential
	err := db.Where("token_hash = ? AND revoked_at IS NULL", hashToken(rawToken)).First(&cred).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	go func() {
		if err := db.Model(&cred).Update("last_used_at", time.Now()).Error; err != nil {
			slog.Warn("failed to update last_used_at", "credential_id", cred.ID, "error", err)
		}
	}()
	var actor Actor
	if err := db.First(&actor, "id = ?", cred.ActorID).Error; err != nil {
		return nil, err
	}
	return &actor, nil
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
		names[a.ID] = a.DisplayName
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

func RevokeAllAgentCredentials(db *gorm.DB, actorID string) error {
	return db.Model(&AgentCredential{}).
		Where("actor_id = ? AND revoked_at IS NULL", actorID).
		Update("revoked_at", time.Now()).Error
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
