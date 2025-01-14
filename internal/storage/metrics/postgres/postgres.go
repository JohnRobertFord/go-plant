package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
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

	for _, i := range [3]int{1, 3, 5} {
		conn, err := pgx.Connect(ctx, p.cfg.DatabaseDsn)
		if err == nil {
			defer func() {
				err := conn.Close(ctx)
				if err != nil {
					log.Printf("[ERR][DB] error while closing conntection: %s\n", err)
				}
			}()
			return nil
		}

		time.Sleep(time.Duration(i) * time.Second)
		log.Printf("Connect to DB. Retry: %d\n", i)
	}
	log.Printf("[ERR][DB] failed to connect DB")
	return fmt.Errorf("[ERR][DB] failed to connect DB")
}

func (p *postgres) SelectAll(ctx context.Context) ([]metrics.Element, error) {

	err := p.check(ctx)
	if err != nil {
		return []metrics.Element{}, err
	}
	rows, err := p.db.Query(ctx, getAllMetricsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	elements, err := pgx.CollectRows(rows, pgx.RowToStructByName[metrics.Element])
	if err != nil {
		log.Printf("CollectRows error: %v", err)
		return nil, err
	}

	return elements, nil
}
func (p *postgres) Insert(ctx context.Context, el metrics.Element) (metrics.Element, error) {

	err := p.check(ctx)
	if err != nil {
		return metrics.Element{}, err
	}
	switch el.MType {
	case "gauge":
		_, err := p.db.Exec(ctx, insertWithConflictQuery, el.ID, el.MType, el.Value, el.Delta, el.Value, el.Delta)
		if err != nil {
			return metrics.Element{}, err
		}
		return el, nil
	case "counter":
		rows, err := p.db.Query(ctx, getOneMetricQuery, el.ID, el.MType)
		if err != nil {
			return metrics.Element{}, err
		}

		var outDelta int64
		ex, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[metrics.Element])
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				log.Printf("CollectRows error: %v, metric: %v", err, el.ID)
				return metrics.Element{}, err
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
			return metrics.Element{}, err
		}
		return out, nil
	default:
		return metrics.Element{}, nil
	}
}

func (p *postgres) Select(ctx context.Context, el metrics.Element) (metrics.Element, error) {

	err := p.check(ctx)
	if err != nil {
		return metrics.Element{}, err
	}
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

func (p *postgres) GetConfig() *config.Config {
	return p.cfg
}

func retry(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {

	var err error
	for _, i := range [3]int{1, 3, 5} {
		dbPool, err := pgxpool.New(ctx, cfg.DatabaseDsn)
		if err == nil {
			return dbPool, nil
		}
		time.Sleep(time.Duration(i) * time.Second)
		log.Printf("Retry: %d\n", i)
	}
	log.Printf("[ERR][DB] failed to connect DB")
	return nil, err
}
func (p *postgres) check(ctx context.Context) error {
	err := p.db.Ping(ctx)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code) {
			newConn, err := retry(ctx, p.cfg)
			p.db = newConn
			return err
		} else {
			return err
		}
	}
	return nil
}
