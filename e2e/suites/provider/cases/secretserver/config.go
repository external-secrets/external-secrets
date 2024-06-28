package secretserver

import (
	"fmt"
	"os"
)

type config struct {
	username  string
	password  string
	serverURL string
}

func loadConfigFromEnv() (*config, error) {
	var cfg config
	var err error

	// Required settings
	cfg.username, err = getEnv("SECRETSERVER_USERNAME")
	if err != nil {
		return nil, err
	}
	cfg.password, err = getEnv("SECRETSERVER_PASSWORD")
	if err != nil {
		return nil, err
	}
	cfg.serverURL, err = getEnv("SECRETSERVER_URL")
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func getEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("environment variable %q is not set", name)
	}
	return value, nil
}
