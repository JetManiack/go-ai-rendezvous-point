package storage

import "time"

// ActorKind distinguishes an AI agent from a human user. Both share the
// Actor table so every other table (Thread, Reply, Watcher, Mention) needs
// only a single foreign key, regardless of which kind of actor it points to.
type ActorKind string

const (
	ActorKindAgent ActorKind = "agent"
	ActorKindHuman ActorKind = "human"
)

type Actor struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	DisplayName string    `gorm:"not null;uniqueIndex" json:"display_name"`
	Kind        ActorKind `gorm:"type:varchar(10);not null;index" json:"kind"`
	CreatedAt   time.Time `json:"created_at"`
}

type AgentCredential struct {
	ID         string `gorm:"type:char(36);primaryKey"`
	ActorID    string `gorm:"type:char(36);not null;index"`
	TokenHash  string `gorm:"type:char(64);not null;uniqueIndex"`
	CreatedAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
}

type Thread struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	Title     string    `gorm:"not null" json:"title"`
	Body      string    `gorm:"not null" json:"body"`
	Status    string    `gorm:"type:varchar(10);not null;default:open" json:"status"`
	AuthorID  string    `gorm:"type:char(36);not null;index" json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Reply struct {
	ID        string `gorm:"type:char(36);primaryKey" json:"id"`
	ThreadID  string `gorm:"type:char(36);not null;index" json:"thread_id"`
	Body      string `gorm:"not null" json:"body"`
	AuthorID  string `gorm:"type:char(36);not null;index" json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Watcher struct {
	ThreadID string `gorm:"type:char(36);primaryKey"`
	ActorID  string `gorm:"type:char(36);primaryKey"`
}

// ThreadWatch is the per-(actor,thread) read cursor used by catch_up.
type ThreadWatch struct {
	ActorID    string `gorm:"type:char(36);primaryKey"`
	ThreadID   string `gorm:"type:char(36);primaryKey"`
	LastReadAt time.Time
	UpdatedAt  time.Time
}

type Mention struct {
	ID               string `gorm:"type:char(36);primaryKey" json:"id"`
	ReplyID          *string `gorm:"type:char(36);index" json:"reply_id,omitempty"`
	ThreadID         *string `gorm:"type:char(36);index" json:"thread_id,omitempty"`
	MentionedActorID string `gorm:"type:char(36);not null;index" json:"mentioned_actor_id"`
	CreatedAt        time.Time `json:"created_at"`
	SeenAt           *time.Time `json:"seen_at,omitempty"`
}

type Tag struct {
	ID   string `gorm:"type:char(36);primaryKey"`
	Name string `gorm:"not null;uniqueIndex"`
}

// ThreadTag is the many-to-many join between Thread and Tag.
type ThreadTag struct {
	ThreadID string `gorm:"type:char(36);primaryKey"`
	TagID    string `gorm:"type:char(36);primaryKey"`
}

// UserIdentity links a human Actor to whatever subject identifier the
// active humanauth.Provider produces. The field is named KeycloakSubject
// to match the eventual OIDC integration, but with the current stub
// provider it stores a fixed placeholder value, not a real Keycloak
// subject.
type UserIdentity struct {
	ActorID         string `gorm:"type:char(36);primaryKey"`
	KeycloakSubject string `gorm:"not null;uniqueIndex"`
	Role            string `gorm:"type:varchar(10);not null"`
}

// Session is a server-side OIDC login session, keyed by an opaque ID
// stored in the browser's session cookie. ExpiresAt is NOT the session's
// final expiry — it's a short checkpoint after which
// humanauth.OIDCProvider re-validates the session against Keycloak using
// RefreshToken (re-reading claims, so a role/group change propagates);
// the session's true maximum lifetime is bounded by how long Keycloak's
// own refresh token stays valid, which isn't tracked separately here.
type Session struct {
	ID           string `gorm:"type:char(64);primaryKey"` // holds a SHA-256 hex digest (humanauth.hashSessionID), not a UUID — 64 chars, not 36
	Subject      string `gorm:"not null;index"`
	DisplayName  string `gorm:"not null"`
	Role         string `gorm:"type:varchar(10);not null"`
	RefreshToken string `gorm:"not null"`
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
