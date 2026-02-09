package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Postgres `yaml:"postgres"`
}

type Postgres struct {
	User     string `yaml:"user"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DBname   string `yaml:"dbname"`
}

func MuslLoad() Config {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	var cfg Config
	err = cleanenv.ReadConfig(os.Getenv("DPM_CONFIG_PATH"), &cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}
