package storage

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type BlobStorage struct {
	BaseDir string
	dbs     map[string]*sql.DB
	mu      sync.RWMutex
}

func NewBlobStorage(baseDir string) (*BlobStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &BlobStorage{
		BaseDir: baseDir,
		dbs:     make(map[string]*sql.DB),
	}, nil
}

func (s *BlobStorage) getAppDBPath(appID string) string {
	return filepath.Join(s.BaseDir, fmt.Sprintf("app_%s.db", appID))
}

func (s *BlobStorage) getDSN(appID string) string {
	return fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL", s.getAppDBPath(appID))
}

func (s *BlobStorage) getDB(appID string) (*sql.DB, error) {
	s.mu.RLock()
	db, ok := s.dbs[appID]
	s.mu.RUnlock()
	if ok {
		return db, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring lock
	if db, ok := s.dbs[appID]; ok {
		return db, nil
	}

	db, err := sql.Open("sqlite3", s.getDSN(appID))
	if err != nil {
		return nil, err
	}

	// Set connection limits for SQLite
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS blobs (
			id INTEGER PRIMARY KEY,
			release_id INTEGER UNIQUE,
			data BLOB
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	s.dbs[appID] = db
	return db, nil
}

func (s *BlobStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lastErr error
	for appID, db := range s.dbs {
		if err := db.Close(); err != nil {
			lastErr = err
		}
		delete(s.dbs, appID)
	}
	return lastErr
}

func (s *BlobStorage) Store(appID string, releaseID uint, reader io.Reader) error {
	db, err := s.getDB(appID)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT OR REPLACE INTO blobs (release_id, data) VALUES (?, ?)", releaseID, data)
	return err
}

func (s *BlobStorage) Get(appID string, releaseID uint) ([]byte, error) {
	db, err := s.getDB(appID)
	if err != nil {
		return nil, err
	}

	var data []byte
	err = db.QueryRow("SELECT data FROM blobs WHERE release_id = ?", releaseID).Scan(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
