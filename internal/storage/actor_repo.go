package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrInvalidToken = errors.New("invalid or revoked token")

func CreateAgent(db *gorm.DB, displayName string) (*Actor, error) {
	actor := &Actor{
		ID:          uuid.NewString(),
		DisplayName: displayName,
		Kind:        ActorKindAgent,
	}
	if err := db.Create(actor).Error; err != nil {
		return nil, err
	}
	return actor, nil
}

// IssueAgentToken generates a new bearer token for actorID and persists
// only its hash. The raw token is returned once and never stored.
func IssueAgentToken(db *gorm.DB, actorID string) (rawToken string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	rawToken = "arp_" + base64.RawURLEncoding.EncodeToString(raw)

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

	now := time.Now()
	if err := db.Model(&cred).Update("last_used_at", now).Error; err != nil {
		slog.Warn("failed to update agent credential last_used_at", "credential_id", cred.ID, "error", err)
	}

	var actor Actor
	if err := db.First(&actor, "id = ?", cred.ActorID).Error; err != nil {
		return nil, err
	}
	return &actor, nil
}

// RevokeAgentToken revokes a single credential by its ID. Revoking a
// credential that's already revoked, or doesn't exist, is a no-op, not
// an error.
func RevokeAgentToken(db *gorm.DB, credentialID string) error {
	return db.Model(&AgentCredential{}).
		Where("id = ? AND revoked_at IS NULL", credentialID).
		Update("revoked_at", time.Now()).Error
}

// RevokeAllAgentCredentials revokes every non-revoked credential belonging
// to actorID — the bulk form of RevokeAgentToken, used when an agent is
// "deleted" via the REST API (which removes its ability to authenticate
// without deleting its Actor row or its thread/reply history).
func RevokeAllAgentCredentials(db *gorm.DB, actorID string) error {
	return db.Model(&AgentCredential{}).
		Where("actor_id = ? AND revoked_at IS NULL", actorID).
		Update("revoked_at", time.Now()).Error
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
