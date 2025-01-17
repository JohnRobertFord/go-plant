package utils

import (
	"context"
	"log"
	"time"
)

func Retry(ctx context.Context, f func() error) error {

	var err error
	for _, i := range [3]int{1, 3, 5} {
		if err := ctx.Err(); err != nil {
			return err
		}
		err = f()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * time.Second)
		log.Printf("Retry func: %d", i)
	}
	return err
}
