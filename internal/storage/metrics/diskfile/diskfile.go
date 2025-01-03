package diskfile

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/JohnRobertFord/go-plant/internal/storage/metrics"
)

func Read4File(ms metrics.Storage) {
	filename := ms.GetConfig().FilePath
	log.Printf("Restore from: %s", filename)
	dataFile, err := os.Open(filename)
	if err != nil {
		log.Printf("cant read from %s", filename)
		return
	}
	var in []metrics.Element
	jsonParser := json.NewDecoder(dataFile)
	if err = jsonParser.Decode(&in); err != nil {
		log.Println("wrong format metric data")
		return
	}

	for _, el := range in {
		if (el.MType == "gauge" && el.Value != nil) || (el.MType == "counter" && el.Delta != nil) {
			ms.Insert(el)
		} else {
			log.Printf("error read \"%s\" metric", el.ID)
			continue
		}
	}
}

func Write2File(ms metrics.Storage) error {

	file, err := os.OpenFile(ms.GetConfig().FilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	var buf []string
	list := ms.SelectAll()
	for _, el := range list {
		switch el.MType {
		case "counter":
			buf = append(buf, fmt.Sprintf(
				"{\"id\":\"%s\",\"type\":\"counter\",\"delta\":%v}", el.ID, *el.Delta))
		case "gauge":
			buf = append(buf, fmt.Sprintf("{\"id\":\"%s\",\"type\":\"gauge\",\"value\":%v}", el.ID, *el.Value))
		default:
			log.Printf("unknown type %s\n", el.MType)
		}
	}
	_, err = fmt.Fprintf(file, "[%s]", strings.Join(buf, ","))
	return err
}
