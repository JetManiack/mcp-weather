package storage

import "time"

type ActorKind string

const ActorKindAgent ActorKind = "agent"

// Actor is an AI agent that holds credentials and appears in the audit log.
type Actor struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	DisplayName string    `gorm:"not null;uniqueIndex" json:"display_name"`
	Kind        ActorKind `gorm:"type:varchar(10);not null;index" json:"kind"`
	CreatedAt   time.Time `json:"created_at"`
}

// AgentCredential is a single bearer token for an agent. The raw token is
// never stored — only its SHA-256 hash is persisted.
type AgentCredential struct {
	ID         string     `gorm:"type:char(36);primaryKey" json:"id"`
	ActorID    string     `gorm:"type:char(36);not null;index" json:"actor_id"`
	TokenHash  string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type ToolCallStatus string

const (
	ToolCallStatusOK    ToolCallStatus = "ok"
	ToolCallStatusError ToolCallStatus = "error"
)

// ToolCall is one recorded MCP tool invocation.
type ToolCall struct {
	ID      string `gorm:"type:char(36);primaryKey" json:"id"`
	ActorID string `gorm:"type:char(36);not null;index:idx_tool_calls_actor_created,priority:1" json:"actor_id"`
	Tool    string `gorm:"type:varchar(64);not null;index:idx_tool_calls_tool_created,priority:1" json:"tool"`
	Args    string `gorm:"type:text;not null" json:"args"`

	Status       ToolCallStatus `gorm:"type:varchar(10);not null;index:idx_tool_calls_status_created,priority:1" json:"status"`
	ErrorMessage string         `gorm:"type:text" json:"error_message,omitempty"`

	DurationMS      int64  `json:"duration_ms"`
	ResponseBytes   int64  `json:"response_bytes"`
	ResponsePreview string `gorm:"type:text" json:"response_preview,omitempty"`
	Truncated       bool   `json:"truncated"`

	CreatedAt time.Time `gorm:"index;index:idx_tool_calls_actor_created,priority:2;index:idx_tool_calls_tool_created,priority:2;index:idx_tool_calls_status_created,priority:2" json:"created_at"`
}
