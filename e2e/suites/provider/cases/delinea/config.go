/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package delinea

import (
	"fmt"
	"os"
)

type config struct {
	tld          string
	urlTemplate  string
	tenant       string
	clientID     string
	clientSecret string
}

func loadConfigFromEnv() (*config, error) {
	var cfg config
	var err error

	// Optional settings
	cfg.tld, _ = getEnv("DELINEA_TLD")
	cfg.urlTemplate, _ = getEnv("DELINEA_URL_TEMPLATE")

	// Required settings
	cfg.tenant, err = getEnv("DELINEA_TENANT")
	if err != nil {
		return nil, err
	}
	cfg.clientID, err = getEnv("DELINEA_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	cfg.clientSecret, err = getEnv("DELINEA_CLIENT_SECRET")
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
