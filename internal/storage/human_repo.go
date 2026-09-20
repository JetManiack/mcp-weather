package storage

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetOrCreateHumanActor provisions an Actor+UserIdentity on first login and
// updates the display name and role on subsequent logins. It is idempotent and
// safe under concurrent calls.
func GetOrCreateHumanActor(db *gorm.DB, subject, displayName, role string) (*Actor, error) {
	var identity UserIdentity
	err := db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("keycloak_subject = ?", subject).First(&identity)
		if res.Error == nil {
			// Already exists — sync display name and role.
			return tx.Model(&identity).Updates(map[string]any{
				"role": role,
			}).Error
		}
		if res.Error != gorm.ErrRecordNotFound {
			return res.Error
		}

		actor := &Actor{
			ID:   uuid.NewString(),
			Name: displayName,
			Kind: ActorKindHuman,
		}
		// Use OnConflict to handle a race where two concurrent first-logins
		// try to create the same display name.
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"kind"}),
		}).Create(actor).Error; err != nil {
			return err
		}
		// If the name conflicted, reload to get the existing ID.
		if err := tx.Where("name = ? AND kind = ?", displayName, ActorKindHuman).First(actor).Error; err != nil {
			return err
		}

		identity = UserIdentity{
			ActorID:         actor.ID,
			KeycloakSubject: subject,
			Role:            role,
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "actor_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"role"}),
		}).Create(&identity).Error
	})
	if err != nil {
		return nil, err
	}

	var actor Actor
	if err := db.First(&actor, "id = ?", identity.ActorID).Error; err != nil {
		return nil, err
	}
	return &actor, nil
}
