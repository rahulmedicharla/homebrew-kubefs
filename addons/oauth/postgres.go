package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, connectionString string) (error, *Postgres) {
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return err, nil
	}
	log.Println("Postgres connected")
	return nil, &Postgres{pool}
}

func(p *Postgres) QueryRow(ctx context.Context, stmt Statement, dest ...any) error {
	row := p.pool.QueryRow(ctx, stmt.Query, stmt.Args...)
	err := row.Scan(dest...)
	if err != nil {
		return err
	}
	return nil
}

func(p *Postgres) Exec(ctx context.Context, stmt Statement) error {
	tx, err := p.pool.Begin(ctx)
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
	p.pool.Close()
}