package sqlite

import (
	"URLShortener/internal/storage"
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS url(
		id INTEGER PRIMARY KEY, 
		alias TEXT NOT NULL UNIQUE,
		url TEXT NOT NULL);
	CREATE INDEX IF NOT EXISTS idx_alias ON url(alias);
`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) SaveURL(urlToSave string, alias string) (int64, error) {
	const op = "storage.sqlite.SaveURL"

	stmt, err := s.db.Prepare("INSERT INTO url(url, alias) VALUES (?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s, %w", op, err)
	}

	res, err := stmt.Exec(urlToSave, alias)
	if err != nil {
		if sqliteErr, ok := errors.AsType[*sqlite.Error](err); ok && sqliteErr.Code() == sqlitelib.SQLITE_CONSTRAINT_UNIQUE {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}
		return 0, fmt.Errorf("%s, %w", op, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get last insert id: %w", op, err)
	}
	return id, nil
}

func (s *Storage) GetURL(alias string) (string, error) {
	const op = "storage.sqlite.GetURL"
	schema := `SELECT url FROM url WHERE alias = ?`

	var url string
	err := s.db.QueryRow(schema, alias).Scan(&url)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("%s, %w", op, storage.ErrURLNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("%s, %w", op, err)
	}

	return url, nil
}
