package lastlogin

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func generateAuditLog(lines int) string {
	var b strings.Builder
	b.Grow(lines * 128)

	hosts := []string{"foo.example.com", "bar.forinstance.net"}
	users := []string{"bob", "alice", "john", "jane", "exporter"}
	clients := []string{"10.95.64.29", "10.98.84.138", "10.0.0.1"}

	ts := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)

	for i := range lines {
		host := hosts[i%len(hosts)]
		user := users[i%len(users)]
		client := clients[i%len(clients)]

		// 1 in 10 lines is a CONNECT
		op := "READ"
		switch i % 10 {
		case 0:
			op = "CONNECT"
		case 1:
			op = "DISCONNECT"
		}

		fmt.Fprintf(&b, "%s,%s,%s,%s,%d,0,%s,,,0\n",
			ts.Format("20060102 15:04:05"),
			host, user, client,
			1000000+i,
			op,
		)
		ts = ts.Add(time.Second)
	}
	return b.String()
}

// Run this bench:
// go test ./internal/lastlogin -bench=BenchmarkParseAuditReader -benchmem -count=10
func BenchmarkParseAuditReader(b *testing.B) {
	cfg := Config{
		TimeFormatAudit: "20060102 15:04:05",
		TimeLocation:    time.UTC,
	}

	// 900k lines ≈ a large but realistic single audit file
	content := generateAuditLog(900000)

	b.ReportAllocs()

	for b.Loop() {
		r := strings.NewReader(content)
		_, err := ParseAuditReader(cfg, r)
		if err != nil {
			b.Fatal(err)
		}
	}
}
