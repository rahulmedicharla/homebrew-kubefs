package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	wpool *pgxpool.Pool
	rpool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, writeConnectionString string, readConnectionString string) (*Postgres, error) {
	wpool, err := pgxpool.New(ctx, writeConnectionString)
	if err != nil {
		return nil, err
	}

	rpool, err := pgxpool.New(ctx, readConnectionString)
	if err != nil {
		return nil, err
	}

	log.Println("Postgres connected")
	return &Postgres{wpool: wpool, rpool: rpool}, err
}

func (p *Postgres) QueryRow(ctx context.Context, stmt Statement, dest ...any) error {
	row := p.rpool.QueryRow(ctx, stmt.Query, stmt.Args...)
	err := row.Scan(dest...)
	if err != nil {
		return err
	}
	return nil
}

func (p *Postgres) Exec(ctx context.Context, stmt Statement) error {
	tx, err := p.wpool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, stmt.Query, stmt.Args...)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Postgres) Close() {
	p.wpool.Close()
	p.rpool.Close()
}
