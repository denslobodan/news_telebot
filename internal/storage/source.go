package storage

// Для чего в методах создаётся нвоое соединение?
// 	conn, err := s.db.Connx(ctx)
// когда можно обращаться к db структуры напрямую!
// s.db.GetContext

import (
	"context"
	"telebot/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/samber/lo"
)

type SourcePostgresStorage struct {
	db *sqlx.DB
}

func (s *SourcePostgresStorage) Sources(ctx context.Context) ([]model.Source, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var sources []dbSource
	if err := conn.SelectContext(ctx, &sources, `SELECT * FROM sources`); err != nil {
		return nil, err
	}

	return lo.Map(sources, func(source dbSource, _ int) model.Source { return model.Source(source) }), nil
}

func (s *SourcePostgresStorage) SourceByID(ctx context.Context, id int64) (*model.Source, error) {
	conn, err := s.db.Connx((ctx))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var source dbSource
	if err := conn.GetContext(ctx, &source, `SELECT * FROM sources WHERE id = $1`, id); err != nil {
		return nil, err
	}
	// приведение типа dbSource к типу model.Source
	// возвращается указатель полуенный source
	return (*model.Source)(&source), nil
}

func (s *SourcePostgresStorage) Add(ctx context.Context, source model.Source) (int64, error) {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	var id int64

	row := conn.QueryRowContext(
		ctx,
		`INSERT INTO sources (name, feed_url, created_at) VALUES ($1, $2, $3) RETURNING id`,
		source.Name,
		source.FeedURL,
		source.CreateAt,
	)
	if err := row.Err(); err != nil {
		return 0, err
	}

	if err := row.Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *SourcePostgresStorage) Delete(ctx context.Context, id int64) error {
	conn, err := s.db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `DELETE FROM sources WHERE id = $1`, id); err != nil {
		return err
	}
	return nil
}

type dbSource struct {
	ID       int64     `db:"id"`
	Name     string    `db:"name"`
	FeedURL  string    `db:"feed_url"`
	CreateAt time.Time `db:"created_at"`
}
