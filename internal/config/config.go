package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AdminToken string
	DBURL      string
	DockTTL    time.Duration
	ServerPort string
	CacheTTL   time.Duration
}

func InitEnv() error {
	err := godotenv.Load("./secrets.env")
	if err != nil {
		return err
	}
	return nil
}

func ItitConfig() (*Config, error) {
	c := &Config{}
	c.AdminToken = os.Getenv("ADMINTOKEN")
	c.ServerPort = os.Getenv("SERVERPORT")
	ttlStr := os.Getenv("TTLSESION")
	if ttlStr == "" {
		return nil, fmt.Errorf("TTLSESION is required")
	}
	ttl, err := strconv.Atoi(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid TTLSESION: %v", err)
	}
	c.DockTTL = time.Second * time.Duration(ttl)
	ttlStr = os.Getenv("TTLCACHE")
	if ttlStr == "" {
		return nil, fmt.Errorf("TTLCACHE is required")
	}
	ttl, err = strconv.Atoi(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid TTLSESION: %v", err)
	}
	c.CacheTTL = time.Second * time.Duration(ttl)
	username := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	dbname := os.Getenv("DBNAME")

	for _, v := range []struct {
		val, name string
	}{
		{username, "USERNAME"},
		{password, "PASSWORD"},
		{host, "HOST"},
		{port, "PORT"},
		{dbname, "DBNAME"},
	} {
		if v.val == "" {
			return nil, fmt.Errorf("%s is required", v.name)
		}
	}

	c.DBURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", username, password, host, port, dbname)

	return c, nil
}
