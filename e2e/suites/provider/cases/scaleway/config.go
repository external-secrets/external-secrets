package scaleway

import (
	"fmt"
	"os"
)

type config struct {
	apiUrl    *string
	region    string
	projectId string
	accessKey string
	secretKey string
}

func loadConfigFromEnv() (*config, error) {

	var cfg config
	var err error

	if apiUrl, ok := os.LookupEnv("SCALEWAY_API_URL"); ok {
		cfg.apiUrl = &apiUrl
	}

	cfg.region, err = getEnv("SCALEWAY_REGION")
	if err != nil {
		return nil, err
	}

	cfg.projectId, err = getEnv("SCALEWAY_PROJECT_ID")
	if err != nil {
		return nil, err
	}

	cfg.accessKey, err = getEnv("SCALEWAY_ACCESS_KEY")
	if err != nil {
		return nil, err
	}

	cfg.secretKey, err = getEnv("SCALEWAY_SECRET_KEY")
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
