package lastlogin

import (
	"database/sql"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"log/slog"
	"os"
	"time"
)

func OpenOrCreateDB(cfg Config) (*sql.DB, error) {
	dbFileExists := true
	_, err := os.Stat(cfg.SqlitePath)
	if os.IsNotExist(err) {
		dbFileExists = false
	}

	db, err := sql.Open("sqlite", cfg.SqlitePath)

	if !dbFileExists {
		err = InitDatabase(db)
		if err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

func InitDatabase(db *sql.DB) error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS Accounts (
		name      TEXT,
		host      TEXT,
		last_seen TIMESTAMP,
		PRIMARY KEY (name, host)
	);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("%q: %s\n", err, sqlStmt)
	}
	return nil
}

func GetLastProcessedTime(cfg Config, db *sql.DB) (time.Time, error) {
	var lastRunDate sql.NullString
	err := db.QueryRow("SELECT last_seen FROM Accounts where name = 'mysql' and host = 'localhost'").Scan(&lastRunDate)

	// Check if lastRunDate is NULL (which will be the case for an empty table)
	if !lastRunDate.Valid || lastRunDate.String == "" {
		slog.Debug("No valid dates found in the Accounts table")
		return time.Date(1, 1, 1, 0, 0, 1, 0, cfg.TimeLocation), nil
	}

	// SQLite stores the date in the format specified by cfg.TimeFormatDB (e.g., "2006-01-02 15:04:05-07:00"),
	// but returns it as RFC3339 (e.g., "2024-10-16T11:15:17.033+02:00") when queried.
	// We parse it using time.RFC3339 to accommodate this behavior.
	t, err := time.Parse(time.RFC3339, lastRunDate.String)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing date '%s' with layout '%s': %w",
			lastRunDate.String, cfg.TimeFormatDB, err)
	}

	return t, nil
}

func UpdateLastProcessedTime(cfg Config, db *sql.DB, lastProcessedTime time.Time) error {
	formattedTime := lastProcessedTime.Format(cfg.TimeFormatDB)

	updateStmt, err := db.Prepare(`
		UPDATE Accounts
		SET last_seen = MAX(last_seen, ?)
		WHERE name = 'mysql' AND host = 'localhost'
	`)
	if err != nil {
		return fmt.Errorf("error preparing update statement: %w", err)
	}
	defer updateStmt.Close()

	// Prepare the insert statement
	insertStmt, err := db.Prepare(`
		INSERT INTO Accounts (name, host, last_seen)
		VALUES ('mysql', 'localhost', ?)
	`)
	if err != nil {
		return fmt.Errorf("error preparing insert statement: %w", err)
	}
	defer insertStmt.Close()

	result, err := updateStmt.Exec(formattedTime)
	if err != nil {
		return fmt.Errorf("error updating last processing date: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	// If no rows were affected, insert a new record
	if rowsAffected == 0 {
		_, err = insertStmt.Exec(formattedTime)
		if err != nil {
			return fmt.Errorf("error inserting last processing date: %w", err)
		}
	}
	return nil
}

func InsertOrUpdateAccounts(cfg Config, db *sql.DB, accounts map[string]AccountInfo) (uint64, uint64, error) {
	var insertedAccounts uint64 = 0
	var updatedAccounts uint64 = 0

	selectStmt, err := db.Prepare(`
		SELECT last_seen
		FROM Accounts
		WHERE name = ?
		AND host = ?
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("error preparing select statement: %w", err)
	}

	// Prepare the update statement
	updateStmt, err := db.Prepare(`
		UPDATE Accounts
		SET last_seen = ?
		WHERE name = ? AND host = ?
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("error preparing update statement: %w", err)
	}
	defer updateStmt.Close()

	// Prepare the insert statement
	insertStmt, err := db.Prepare(`
		INSERT INTO Accounts (name, host, last_seen)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("error preparing insert statement: %w", err)
	}
	defer insertStmt.Close()

	for _, a := range accounts {
		newTime := time.Date(
			a.LastSeen.Year(), a.LastSeen.Month(), a.LastSeen.Day(),
			a.LastSeen.Hour(), a.LastSeen.Minute(), a.LastSeen.Second(),
			a.LastSeen.Nanosecond(), cfg.TimeLocation)
		formatted := newTime.Format(cfg.TimeFormatDB)

		var existing sql.NullString
		err := selectStmt.QueryRow(a.Name, a.Host).Scan(&existing)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// brand-new account -> must insert
			if _, err := insertStmt.Exec(a.Name, a.Host, formatted); err != nil {
				return 0, 0, fmt.Errorf("error inserting account: %w", err)
			}
			insertedAccounts++
		case err != nil:
			return 0, 0, fmt.Errorf("error selecting account: %w", err)
		default:
			stored, err := time.Parse(time.RFC3339, existing.String)
			if err != nil {
				return 0, 0, fmt.Errorf("error parsing stored time %q: %w", existing.String, err)
			}
			if newTime.After(stored) { // true instant comparison, DST-safe
				if _, err := updateStmt.Exec(formatted, a.Name, a.Host); err != nil {
					return 0, 0, fmt.Errorf("error updating account: %w", err)
				}
				updatedAccounts++
			}
		}
	}

	return insertedAccounts, updatedAccounts, nil
}
