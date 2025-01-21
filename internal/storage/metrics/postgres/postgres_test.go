package postgres

import (
	"log"
	"testing"

	"github.com/ory/dockertest/v3"
)

func TestPG(t *testing.T) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	resource, err := pool.Run(
		"postgres", "11",
		[]string{
			"POSTGRES_DB=metrics_test",
			"POSTGRES_PASSWORD=s3cr3t",
		},
	)
	if err != nil {
		log.Panicf("pool.Run failed run docker container: %v", err)
	}
	defer func(err error) {
		if err != nil {
			if e := pool.Purge(resource); e != nil {
				log.Printf("failed to purge the postgres container: %v", e)
			}
		}
	}(err)

}
