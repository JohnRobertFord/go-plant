package diskfile

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
)

func Read4File(ctx context.Context, ms metrics.Storage) error {
	filename := ms.GetConfig().FilePath
	log.Printf("Restore from: %s", filename)
	dataFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	var in []metrics.Element
	jsonParser := json.NewDecoder(dataFile)
	if err = jsonParser.Decode(&in); err != nil {
		return err
	}

	for _, el := range in {
		if (el.MType == "gauge" && el.Value != nil) || (el.MType == "counter" && el.Delta != nil) {
			ms.Insert(ctx, el)
		} else {
			log.Printf("error read \"%s\" metric", el.ID)
			continue
		}
	}
	return nil
}

func Write2File(ctx context.Context, ms metrics.Storage) error {

	file, err := os.OpenFile(ms.GetConfig().FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	var buf []string
	list, err := ms.SelectAll(ctx)
	if err != nil {
		log.Println(err)
	}
	for _, el := range list {
		switch el.MType {
		case "counter":
			// check to prevent 'panic: runtime error: invalid memory address or nil pointer dereference'
			// when program quits
			if el.Delta != nil {
				buf = append(buf, fmt.Sprintf(
					"{\"id\":\"%s\",\"type\":\"counter\",\"delta\":%v}", el.ID, *el.Delta))
			}
		case "gauge":
			// check to prevent 'panic ...'
			if el.Value != nil {
				buf = append(buf, fmt.Sprintf("{\"id\":\"%s\",\"type\":\"gauge\",\"value\":%v}", el.ID, *el.Value))
			}
		default:
			log.Printf("unknown type %s\n", el.MType)
		}
	}
	_, err = fmt.Fprintf(file, "[%s]", strings.Join(buf, ","))
	return err
}
