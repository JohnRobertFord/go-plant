package postgres

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
	"github.com/JohnRobertFord/go-plant/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	createTableQuery = `CREATE TABLE IF NOT EXISTS metrics(
	"id" int generated always as identity,
	"name" varchar(255) UNIQUE,
	"type" varchar(50),
	"value" double precision,
	"delta" bigint
	);`
	insertWithConflictQuery = `INSERT INTO metrics(name, type, value, delta) VALUES($1,$2,$3,$4) ON CONFLICT (name) DO UPDATE SET value=$5, delta=$6 ;`
	getAllMetricsQuery      = `SELECT name, type, value, delta FROM metrics;`
	getOneMetricQuery       = `SELECT name, type, value, delta FROM metrics WHERE name=$1 AND type=$2;`
)

type postgres struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

var (
	pgInstance *postgres
	pgOnce     sync.Once
)

func (p *postgres) Close() {
	p.db.Close()
}

func NewPostgresStorage(c *config.Config) (*postgres, error) {

	ctx := context.Background()
	pgOnce.Do(func() {
		dbPool, err := pgxpool.New(ctx, c.DatabaseDsn)
		if err != nil {
			log.Printf("unable to create connection pool: %s", err)
			return
		}
		pgInstance = &postgres{dbPool, c}
	})
	_, err := pgInstance.db.Exec(ctx, createTableQuery)
	return pgInstance, err
}

func (p *postgres) Ping(ctx context.Context) error {

	err := p.db.Ping(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (p *postgres) SelectAll(ctx context.Context) (*[]metrics.Element, error) {

	var out []metrics.Element
	var err error
	utils.Retry(ctx, func() error {
		rows, err := p.db.Query(ctx, getAllMetricsQuery)
		if err != nil {
			return err
		}
		defer rows.Close()
		out, err = pgx.CollectRows(rows, pgx.RowToStructByName[metrics.Element])
		if err != nil {
			log.Printf("CollectRows error: %v", err)
			return err
		}
		return nil
	})

	return &out, err
}
func (p *postgres) Insert(ctx context.Context, el metrics.Element) (*metrics.Element, error) {

	var out metrics.Element
	utils.Retry(ctx, func() error {
		switch el.MType {
		case "gauge":
			_, err := p.db.Exec(ctx, insertWithConflictQuery, el.ID, el.MType, el.Value, el.Delta, el.Value, el.Delta)
			if err != nil {
				return err
			}
			return nil
		case "counter":
			rows, err := p.db.Query(ctx, getOneMetricQuery, el.ID, el.MType)
			if err != nil {
				return err
			}

			var outDelta int64
			ex, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[metrics.Element])
			if err != nil {
				if !errors.Is(err, pgx.ErrNoRows) {
					log.Printf("CollectRows error: %v, metric: %v", err, el.ID)
					return err
				}
				outDelta = *el.Delta
			} else {
				outDelta = *ex.Delta + *el.Delta
			}

			out := metrics.Element{
				ID:    el.ID,
				MType: el.MType,
				Delta: &outDelta,
			}

			_, err = p.db.Exec(ctx, insertWithConflictQuery, out.ID, out.MType, out.Value, out.Delta, out.Value, out.Delta)
			if err != nil {
				log.Println(err)
				return err
			}
			return nil
		}
		return nil
	})
	return &out, nil
}

func (p *postgres) Select(ctx context.Context, el metrics.Element) (*metrics.Element, error) {

	var out metrics.Element

	e := utils.Retry(ctx, func() error {
		row, err := p.db.Query(ctx, getOneMetricQuery, el.ID, el.MType)
		if err != nil {
			return err
		}
		out, err = pgx.CollectOneRow(row, pgx.RowToStructByName[metrics.Element])
		if err != nil {
			return err
		}
		return err
	})

	if e != nil {
		return nil, e
	}

	return &out, nil
}

func (p *postgres) GetConfig() *config.Config {
	return p.cfg
}
