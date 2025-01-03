package postgres

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
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

func NewPostgresStorage(c *config.Config) *postgres {
	ctx := context.Background()
	pgOnce.Do(func() {
		dbPool, err := pgxpool.New(ctx, c.DatabaseDsn)
		if err != nil {
			log.Printf("unable to create connection pool: %v", err)
			return
		}
		pgInstance = &postgres{dbPool, c}
	})
	pgInstance.db.Exec(ctx, createTableQuery)

	return pgInstance
}

func (p *postgres) SelectAll() []metrics.Element {

	rows, err := p.db.Query(context.Background(), getAllMetricsQuery)
	if err != nil {
		return nil
	}
	defer rows.Close()

	elements, err := pgx.CollectRows(rows, pgx.RowToStructByName[metrics.Element])
	if err != nil {
		log.Printf("CollectRows error: %v\n", err)
		return nil
	}

	return elements
}
func (p *postgres) Insert(el metrics.Element) metrics.Element {
	ctx := context.Background()
	switch el.MType {
	case "gauge":
		_, err := p.db.Exec(ctx, insertWithConflictQuery, el.ID, el.MType, el.Value, el.Delta, el.Value, el.Delta)
		if err != nil {
			log.Println(err)
			return metrics.Element{}
		}
		return el
	case "counter":
		rows, err := p.db.Query(ctx, getOneMetricQuery, el.ID, el.MType)
		if err != nil {
			return metrics.Element{}
		}

		var outDelta int64
		ex, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[metrics.Element])
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("CollectRows error: %v, metric: %v", err, el.ID)
				return metrics.Element{}
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
			return metrics.Element{}
		}
		return out
	default:
		return metrics.Element{}
	}
}

func (p *postgres) Select(el metrics.Element) (metrics.Element, error) {
	ctx := context.Background()
	row, err := p.db.Query(ctx, getOneMetricQuery, el.ID, el.MType)
	if err != nil {
		return metrics.Element{}, err
	}

	out, err := pgx.CollectOneRow(row, pgx.RowToStructByName[metrics.Element])
	if err != nil {
		return metrics.Element{}, err
	}

	return out, nil
}

func (p *postgres) GetConfig() config.Config {
	return *p.cfg
}
