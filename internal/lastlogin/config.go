package lastlogin

import (
	"fmt"
	"gopkg.in/ini.v1"
	"log/slog"
	"time"
)

func InitConfig() (Config, error) {

	cfg, err := ini.Load("/etc/mariadb-lastlogin/config.ini")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load ini file: %w", err)
	}

	// Get the default section
	defaultSection := cfg.Section("default")

	// Load TimeLocation
	timeLocationStr := defaultSection.Key("TimeLocation").MustString("Europe/Zurich")
	loc, err := time.LoadLocation(timeLocationStr)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load location: %w", err)
	}

	// Parse LogLevel
	logLevelStr := defaultSection.Key("LogLevel").MustString("ERROR")
	logLevel, err := parseLogLevel(logLevelStr)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse log level: %w", err)
	}

	// Parse MaxWorkers with a default value of 4
	maxWorkers := defaultSection.Key("MaxWorkers").MustInt(4)

	return Config{
		AuditLogFile:    defaultSection.Key("AuditLogFile").MustString("server_audit.log"),
		AuditLogPath:    defaultSection.Key("AuditLogPath").MustString("/var/lib/mysql"),
		SqlitePath:      defaultSection.Key("SqlitePath").MustString("/var/lib/mysql/audit.sqlite"),
		TimeFormatAudit: defaultSection.Key("TimeFormatAudit").MustString("20060102 15:04:05"),
		TimeFormatDB:    defaultSection.Key("TimeFormatDB").MustString("2006-01-02 15:04:05-07:00"),
		TimeLocation:    loc,
		MaxWorkers:      maxWorkers,
		LogLevel:        logLevel,
	}, nil
}

func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level: %s", level)
	}
}
