package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

func Open(path string) (*Store, error) {
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	_, e = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; CREATE TABLE IF NOT EXISTS config (kind TEXT NOT NULL,id TEXT NOT NULL,body BLOB NOT NULL,PRIMARY KEY(kind,id)); CREATE TABLE IF NOT EXISTS snapshots (id TEXT PRIMARY KEY,created_at TEXT NOT NULL,body BLOB NOT NULL);`)
	if e != nil {
		db.Close()
		return nil, e
	}
	return &Store{db}, nil
}
func (s *Store) Put(kind, id string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	_, e = s.DB.Exec("INSERT INTO config(kind,id,body) VALUES(?,?,?) ON CONFLICT(kind,id) DO UPDATE SET body=excluded.body", kind, id, b)
	return e
}
func (s *Store) Delete(kind, id string) error {
	_, e := s.DB.Exec("DELETE FROM config WHERE kind=? AND id=?", kind, id)
	return e
}
func (s *Store) List(kind string, v any) error {
	rows, e := s.DB.Query("SELECT body FROM config WHERE kind=? ORDER BY id", kind)
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return e
		}
		out = append(out, b)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	b, _ := json.Marshal(out)
	return json.Unmarshal(b, v)
}
func (s *Store) Snapshot(id, at string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec("INSERT INTO snapshots VALUES(?,?,?)", id, at, b); e != nil {
		return e
	}
	_, e = tx.Exec("DELETE FROM snapshots WHERE id NOT IN (SELECT id FROM snapshots ORDER BY created_at DESC LIMIT 200)")
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Snapshots(limit int, v any) error {
	if limit < 1 || limit > 200 {
		return fmt.Errorf("invalid history limit")
	}
	rows, e := s.DB.Query("SELECT body FROM snapshots ORDER BY created_at DESC LIMIT ?", limit)
	if e != nil {
		return e
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return e
		}
		out = append(out, b)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	b, _ := json.Marshal(out)
	return json.Unmarshal(b, v)
}
