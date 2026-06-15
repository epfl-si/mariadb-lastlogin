package lastlogin

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func generateTestAccounts(n int) map[string]AccountInfo {
	accounts := make(map[string]AccountInfo, n)
	for i := range n {
		name := fmt.Sprintf("user%d", i)
		host := fmt.Sprintf("host%d", i%50)
		accounts[name+"@"+host] = AccountInfo{
			Name:     name,
			Host:     host,
			LastSeen: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
		}
	}
	return accounts
}

// Run this bench:
// go test ./internal/lastlogin -bench=BenchmarkInsertOrUpdateAccounts -benchmem -count=10
func BenchmarkInsertOrUpdateAccounts(b *testing.B) {
	cfg := Config{
		TimeLocation: time.UTC,
		TimeFormatDB: "2006-01-02 15:04:05-07:00",
	}

	// 500 accounts is a realistic batch size for a busy audit run
	batchSize := 500
	accounts := generateTestAccounts(batchSize)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		// Stop timer around per-iteration DB setup so we only measure the function
		b.StopTimer()

		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			b.Fatal(err)
		}
		if err := InitDatabase(db); err != nil {
			b.Fatal(err)
		}

		// Pre-seed half the accounts so the benchmark exercises updates, not just inserts
		seed := generateTestAccounts(batchSize / 2)
		if _, _, err := InsertOrUpdateAccounts(cfg, db, seed); err != nil {
			b.Fatal(err)
		}

		b.StartTimer()

		_, _, err = InsertOrUpdateAccounts(cfg, db, accounts)
		if err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		db.Close()
		b.StartTimer()
	}
}
