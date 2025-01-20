package metrics

import (
	"context"
	"strconv"

	"github.com/JohnRobertFord/go-plant/internal/config"
)

type Element struct {
	ID    string   `json:"id" db:"name"`
	MType string   `json:"type" db:"type"`
	Delta *int64   `json:"delta,omitempty" db:"delta"`
	Value *float64 `json:"value,omitempty" db:"value"`
}

type Storage interface {
	Insert(context.Context, Element) (*Element, error)
	Select(context.Context, Element) (*Element, error)
	SelectAll(context.Context) (*[]Element, error)
	Ping(context.Context) error
	GetConfig() *config.Config
}

func IsCounter(input string) bool {
	if _, err := strconv.ParseInt(input, 10, 64); err == nil {
		return true
	}
	return false
}

func IsGauge(input string) bool {
	if _, err := strconv.ParseFloat(input, 64); err == nil {
		return true
	}
	return false
}
