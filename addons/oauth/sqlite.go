package main

import (
	"database/sql"
	"log"
	"context"
	_ "github.com/glebarez/sqlite"
)

type SQLite struct {
	db *sql.DB
}

func NewSQLite() (error, *SQLite) {
	db, err := sql.Open("sqlite", "/app/store/sqlite.db")
	if err != nil {
		return err, nil
	}
	log.Println("SQLite connected")
	return nil, &SQLite{db}
}

func (s *SQLite) QueryRow(ctx context.Context, stmt Statement, dest ...interface{}) error {
	row := s.db.QueryRow(stmt.Query, stmt.Args...)
	err := row.Scan(dest...)
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLite) Exec(ctx context.Context, stmt Statement) error {
	_, err := s.db.Exec(stmt.Query, stmt.Args...)
	if err != nil {
		return err
	}

	return nil
}

func (s *SQLite) Close() {
	s.db.Close()
}