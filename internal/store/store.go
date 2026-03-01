package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("creating data directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	if err := setPragmas(db); err != nil {
		db.Close()
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func setPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("setting pragma %q: %w", p, err)
		}
	}
	return nil
}

func (s *Store) migrate() error {
	data, err := migrations.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return fmt.Errorf("reading migration: %w", err)
	}

	// database/sql.Exec only runs the first statement, so we need a
	// single connection and execute statements individually.
	conn, err := s.db.Conn(context.Background())
	if err != nil {
		return fmt.Errorf("getting connection: %w", err)
	}
	defer conn.Close()

	for _, stmt := range splitSQL(string(data)) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := conn.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("executing migration statement: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}

// splitSQL splits a SQL script on semicolons, respecting BEGIN...END blocks (triggers).
func splitSQL(sql string) []string {
	var stmts []string
	var current strings.Builder
	inBlock := false

	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}

		upper := strings.ToUpper(trimmed)
		if strings.Contains(upper, " BEGIN") || strings.HasPrefix(upper, "BEGIN") {
			inBlock = true
		}

		current.WriteString(line)
		current.WriteString("\n")

		if inBlock {
			if strings.HasPrefix(upper, "END;") || strings.HasSuffix(upper, "END;") {
				inBlock = false
				stmts = append(stmts, current.String())
				current.Reset()
			}
		} else if strings.HasSuffix(trimmed, ";") {
			stmts = append(stmts, current.String())
			current.Reset()
		}
	}
	if s := strings.TrimSpace(current.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}
