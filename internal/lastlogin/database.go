package lastlogin

import (
	"database/sql"
	"fmt"
	"log/slog"
	_ "modernc.org/sqlite"
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
	// --- 1. Load existing accounts into memory (ONE query) ---
	existing := make(map[string]time.Time)
	rows, err := db.Query("SELECT name, host, last_seen FROM Accounts")
	if err != nil {
		return 0, 0, fmt.Errorf("error selecting accounts: %w", err)
	}
	for rows.Next() {
		var name, host string
		var lastSeen sql.NullString
		if err := rows.Scan(&name, &host, &lastSeen); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("error scanning account: %w", err)
		}
		stored, err := time.Parse(time.RFC3339, lastSeen.String)
		if err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("error parsing stored time %q: %w", lastSeen.String, err)
		}
		existing[name+"@"+host] = stored
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	// --- 2. Batch all writes in ONE transaction ---
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer tx.Rollback() // safe no-op after Commit()

	updateStmt, err := tx.Prepare(`
		UPDATE Accounts SET last_seen = ? WHERE name = ? AND host = ?
	`)
	if err != nil {
		return 0, 0, err
	}
	defer updateStmt.Close()

	insertStmt, err := tx.Prepare(`
		INSERT INTO Accounts (name, host, last_seen) VALUES (?, ?, ?)
	`)
	if err != nil {
		return 0, 0, err
	}
	defer insertStmt.Close()

	var inserted, updated uint64
	for _, a := range accounts {
		newTime := time.Date(
			a.LastSeen.Year(), a.LastSeen.Month(), a.LastSeen.Day(),
			a.LastSeen.Hour(), a.LastSeen.Minute(), a.LastSeen.Second(),
			a.LastSeen.Nanosecond(), cfg.TimeLocation)
		formatted := newTime.Format(cfg.TimeFormatDB)

		key := a.Name + "@" + a.Host
		if stored, ok := existing[key]; ok {
			if newTime.After(stored) {
				if _, err := updateStmt.Exec(formatted, a.Name, a.Host); err != nil {
					return 0, 0, fmt.Errorf("error updating account: %w", err)
				}
				updated++
			}
		} else {
			if _, err := insertStmt.Exec(a.Name, a.Host, formatted); err != nil {
				return 0, 0, fmt.Errorf("error inserting account: %w", err)
			}
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("error committing transaction: %w", err)
	}

	return inserted, updated, nil
}
