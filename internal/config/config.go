package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Bind          string `json:"bind" env:"ADDRESS"`
	StoreInterval int    `json:"storeInterval" env:"STORE_INTERVAL"`
	FilePath      string `json:"filePath" env:"FILE_STORAGE_PATH"`
	Restore       bool   `json:"isRestored" env:"RESTORE"`
	DatabaseDsn   string `json:"databaseDsn" env:"DATABASE_DSN"`
	Key           string `json:"key" env:"KEY"`
}

func (c *Config) String() string {
	return fmt.Sprintf("[Config] Host:%s, StoreInterval:%v, FilePath:%s, Restore:%t, DatabaseDsn:%s, Key:%s",
		c.Bind,
		c.StoreInterval,
		c.FilePath,
		c.Restore,
		c.DatabaseDsn,
		c.Key)
}

func InitConfig() (*Config, error) {
	var cfg Config
	var envCfg Config

	flag.StringVar(&cfg.Bind, "a", ":8080", "адрес сервера (env ADDRESS)")
	flag.IntVar(&cfg.StoreInterval, "i", 10, "интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск (по умолчанию 300 секунд, значение 0 делает запись синхронной)")
	flag.StringVar(&cfg.FilePath, "f", "metrics.log", "путь файла, для сохранения текущих значений")
	flag.BoolVar(&cfg.Restore, "r", true, "булево значение (true/false), определяющее, загружать или нет ранее сохранённые значения из указанного файла при старте сервера")
	flag.StringVar(&cfg.DatabaseDsn, "d", "", "адрес подключения к БД (env DATABASE_DSN)")
	flag.StringVar(&cfg.Key, "k", "", "ключ для создания подписи заголовков (env KEY)")

	flag.Parse()

	if err := env.Parse(&envCfg); err != nil {
		return nil, fmt.Errorf("cant load config: %e", err)
	}

	if os.Getenv("ADDRESS") != "" {
		cfg.Bind = envCfg.Bind
	}
	if os.Getenv("FILE_STORAGE_PATH") != "" {
		cfg.FilePath = envCfg.FilePath
	}
	if os.Getenv("STORE_INTERVAL") != "" {
		cfg.StoreInterval = envCfg.StoreInterval
	}
	if os.Getenv("RESTORE") != "" {
		cfg.Restore = envCfg.Restore
	}
	if os.Getenv("DATABASE_DSN") != "" {
		cfg.DatabaseDsn = envCfg.DatabaseDsn
	}
	if os.Getenv("KEY") != "" {
		cfg.Key = envCfg.Key
	}

	return &cfg, nil
}
