package cache

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/JohnRobertFord/go-plant/internal/config"
	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
)

type (
	// gauge   float64
	// counter int64

	MemStorage struct {
		mapa map[string]any
		mu   sync.Mutex
		cfg  *config.Config
	}
)

func NewMemStorage(c *config.Config) *MemStorage {
	return &MemStorage{
		mapa: make(map[string]any),
		cfg:  c,
	}
}

func (m *MemStorage) GetConfig() *config.Config {
	return m.cfg
}
func (m *MemStorage) Ping(context.Context) error {
	return fmt.Errorf("no support storage")
}
func (m *MemStorage) Insert(ctx context.Context, el metrics.Element) (metrics.Element, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := metrics.Element{
		ID:    el.ID,
		MType: el.MType,
	}
	switch el.MType {
	case "gauge":
		m.mapa[el.ID] = *el.Value
		out.Value = el.Value
	case "counter":
		if c, ok := m.mapa[el.ID].(int64); ok {
			c += *el.Delta
			m.mapa[el.ID] = c
			out.Delta = &c
		} else {
			m.mapa[el.ID] = *el.Delta
			out.Delta = el.Delta
		}
	default:
		return metrics.Element{}, fmt.Errorf("[ERR][INSERT] cant insert metric %v", el)
	}
	return out, nil
}

func (m *MemStorage) Select(ctx context.Context, el metrics.Element) (metrics.Element, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := metrics.Element{
		ID:    el.ID,
		MType: el.MType,
	}

	if el.MType == "gauge" {
		if f, ok := m.mapa[el.ID].(float64); ok {
			out.Value = &f
		} else {
			return metrics.Element{}, fmt.Errorf("metric not found")
		}
	} else if el.MType == "counter" {
		if c, ok := m.mapa[el.ID].(int64); ok {
			out.Delta = &c
		} else {
			return metrics.Element{}, fmt.Errorf("metric not found")
		}
	}

	return out, nil
}

func (m *MemStorage) SelectAll(ctx context.Context) ([]metrics.Element, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []metrics.Element
	var err error

	for k, v := range m.mapa {
		temp := metrics.Element{
			ID: k,
		}
		switch vt := v.(type) {
		case int64:
			if c, ok := v.(int64); ok {
				temp.MType = "counter"
				temp.Delta = &c
			} else {
				log.Println("error getting counter")
				err = fmt.Errorf("error getting metrics")
			}
		case float64:
			if f, ok := v.(float64); ok {
				temp.MType = "gauge"
				temp.Value = &f
			} else {
				log.Println("error getting gauge")
				err = fmt.Errorf("error getting metrics")
			}
		default:
			log.Printf("unknown type: %v", vt)
		}
		list = append(list, temp)
	}
	return list, err
}
