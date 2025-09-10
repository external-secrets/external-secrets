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
