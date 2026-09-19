package storage

import "time"

type ActorKind string

const ActorKindAgent ActorKind = "agent"

// Actor is an AI agent that holds credentials and appears in the audit log.
type Actor struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	Name      string    `gorm:"not null;uniqueIndex" json:"name"`
	Kind      ActorKind `gorm:"type:varchar(10);not null;index" json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

// AgentCredential is a single bearer token for an agent. The raw token is
// never stored — only its SHA-256 hash is persisted.
type AgentCredential struct {
	ID         string     `gorm:"type:char(36);primaryKey" json:"id"`
	ActorID    string     `gorm:"type:char(36);not null;index" json:"actor_id"`
	Label      string     `gorm:"type:varchar(255)" json:"label,omitempty"`
	TokenHash  string     `gorm:"type:char(64);not null;uniqueIndex" json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// ToolCall is one recorded MCP tool invocation.
type ToolCall struct {
	ID      string `gorm:"type:char(36);primaryKey" json:"id"`
	ActorID string `gorm:"type:char(36);not null;index:idx_tool_calls_actor_called,priority:1" json:"actor_id"`
	Tool    string `gorm:"type:varchar(64);not null;index:idx_tool_calls_tool_called,priority:1" json:"tool"`

	InputJSON string `gorm:"type:text;not null" json:"input_json"`
	IsError   bool   `gorm:"not null;index:idx_tool_calls_error_called,priority:1" json:"is_error"`

	DurationMS int    `json:"duration_ms"`
	OutputSize int    `json:"output_size"`
	OutputJSON string `gorm:"type:text" json:"output_json,omitempty"`
	Truncated  bool   `json:"truncated"`

	CalledAt time.Time `gorm:"index;index:idx_tool_calls_actor_called,priority:2;index:idx_tool_calls_tool_called,priority:2;index:idx_tool_calls_error_called,priority:2" json:"called_at"`
}
