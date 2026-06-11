package lastlogin

import (
	"log/slog"
	"time"
)

type Config struct {
	AuditLogPath    string
	SqlitePath      string
	TimeFormatAudit string
	TimeFormatDB    string
	TimeLocation    *time.Location
	MaxWorkers      int
	LogLevel        slog.Level
}

type FileInfo struct {
	Path    string
	ModTime time.Time
}

type AccountInfo struct {
	Name     string
	Host     string
	LastSeen time.Time
}

type ChunkResult struct {
	Accounts map[string]AccountInfo // key is "username@hostname"
}
