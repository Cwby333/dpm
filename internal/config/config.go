package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Postgres `yaml:"postgres"`
	JWT      `yaml:"jwt"`
	S3       `yaml:"s3"`
}

type Postgres struct {
	User     string `yaml:"user"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DBname   string `yaml:"dbname"`
}

type JWT struct {
	Key string `yaml:"secret-key"`
}

type S3 struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Region    string `yaml:"region"`
	Endpoint  string `yaml:"endpoint"`
	Bucket    string `yaml:"bucket"`
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
