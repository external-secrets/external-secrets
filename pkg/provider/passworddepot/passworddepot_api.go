/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package passworddepot

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DoRequestError = "error: do request: %w"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type AccessData struct {
	ClientID    string `json:"client_id"`
	AccessToken string `json:"access_token"`
}

type Databases struct {
	Databases []struct {
		Name         string    `json:"name"`
		Fingerprint  string    `json:"fingerprint"`
		Date         time.Time `json:"date"`
		Rights       string    `json:"rights"`
		Reasondelete string    `json:"reasondelete"`
	} `json:"databases"`
	Infoclasses          string `json:"infoclasses"`
	Policyforce          string `json:"policyforce"`
	Policyminlength      string `json:"policyminlength"`
	Policyincludeatleast string `json:"policyincludeatleast"`
	Policymingroups      string `json:"policymingroups"`
	Policyselectedgroups string `json:"policyselectedgroups"`
}

type DatabaseEntries struct {
	Name         string  `json:"name"`
	Parent       string  `json:"parent"`
	Entries      []Entry `json:"entries"`
	Infoclasses  string  `json:"infoclasses"`
	Reasondelete string  `json:"reasondelete"`
}

type Entry struct {
	Name        string    `json:"name"`
	Login       string    `json:"login"`
	Password    string    `json:"pass"`
	URL         string    `json:"url"`
	Importance  string    `json:"importance"`
	Date        time.Time `json:"date"`
	Icon        string    `json:"icon"`
	Secondeye   string    `json:"secondeye"`
	Fingerprint string    `json:"fingerprint"`
	Rights      string    `json:"rights"`
	Itemclass   string    `json:"itemclass"`
}

type API struct {
	client   HTTPClient
	baseURL  string
	hostPort string
	secret   *AccessData
	password string
	username string
}

type SecretEntry struct {
	Name        string    `json:"name"`
	Fingerprint string    `json:"fingerprint"`
	Itemclass   string    `json:"itemclass"`
	Login       string    `json:"login"`
	Pass        string    `json:"pass"`
	URL         string    `json:"url"`
	Importance  string    `json:"importance"`
	Date        time.Time `json:"date"`
	Comment     string    `json:"comment"`
	Expirydate  string    `json:"expirydate"`
	Tags        string    `json:"tags"`
	Author      string    `json:"author"`
	Category    string    `json:"category"`
	Icon        string    `json:"icon"`
	Secondeye   string    `json:"secondeye"`
	Secondpass  string    `json:"secondpass"`
	Template    string    `json:"template"`
	Acm         string    `json:"acm"`
	Paramstr    string    `json:"paramstr"`
	Loginid     string    `json:"loginid"`
	Passid      string    `json:"passid"`
	Donotaddon  string    `json:"donotaddon"`
	Markassafe  string    `json:"markassafe"`
	Safemode    string    `json:"safemode"`
}

var errDBNotFound = errors.New("database not found")
var errSecretNotFound = errors.New("secret not found")

// load tls certificates

func NewAPI(ctx context.Context, baseURL, username, password, hostPort string) (*API, error) {
	api := &API{
		baseURL:  baseURL,
		hostPort: hostPort,
		username: username,
		password: password,
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}

	api.client = &http.Client{Transport: tr}
	err := api.login(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	return api, nil
}

func (api *API) doAuthenticatedRequest(r *http.Request) (*http.Response, error) {
	r.Header.Add("access_token", api.secret.AccessToken)
	r.Header.Add("client_id", api.secret.ClientID)

	return api.client.Do(r)
}
func (api *API) getDatabaseFingerprint(database string) (string, error) {
	databases, err := api.ListDatabases()
	if err != nil {
		return "", fmt.Errorf("error: getting database list: %w", err)
	}

	for _, db := range databases.Databases {
		if strings.Contains(db.Name, database) {
			return db.Fingerprint, nil
		}
	}

	return "", errDBNotFound
}

func (api *API) getSecretFingerprint(databaseFingerprint, secretName, folder string) (string, error) {
	secrets, err := api.ListSecrets(databaseFingerprint, folder)
	if err != nil {
		return "", fmt.Errorf("error: getting secrets list: %w", err)
	}

	parts := strings.Split(secretName, ".")
	searchName := parts[0]
	var fingerprint string
	for _, entry := range secrets.Entries {
		if strings.Contains(entry.Name, searchName) {
			fingerprint = entry.Fingerprint
			if len(parts) > 1 {
				return api.getSecretFingerprint(databaseFingerprint, strings.Join(parts[1:], "."), fingerprint)
			}
			return fingerprint, nil
		}
	}
	return "", errSecretNotFound
}

func (api *API) getendpointURL(endpoint string) string {
	return fmt.Sprintf("https://%s:%s/v1.0/%s", api.baseURL, api.hostPort, endpoint)
}

func (api *API) login(ctx context.Context) error {
	loginRequest, err := http.NewRequestWithContext(ctx, "GET", api.getendpointURL("login"), http.NoBody)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	loginRequest.Header.Add("user", api.username)
	loginRequest.Header.Add("pass", api.password)

	resp, err := api.client.Do(loginRequest) //nolint:bodyclose // linters bug
	if err != nil {
		return fmt.Errorf(DoRequestError, err)
	}

	accessData := AccessData{}
	err = ReadAndUnmarshal(resp, &accessData)
	if err != nil {
		return fmt.Errorf("error: failed to unmarshal response body: %w", err)
	}

	api.secret = &accessData

	return nil
}

func (api *API) ListSecrets(dbFingerprint, folder string) (DatabaseEntries, error) {
	endpointURL := api.getendpointURL(fmt.Sprintf("list?db=%s", dbFingerprint))
	if folder != "" {
		endpointURL = fmt.Sprintf("%s&folder=%s", endpointURL, folder)
	}
	listSecrets, err := http.NewRequest("GET", endpointURL, http.NoBody)
	if err != nil {
		return DatabaseEntries{}, fmt.Errorf("error: creating secrets request: %w", err)
	}

	respSecretsList, err := api.doAuthenticatedRequest(listSecrets) //nolint:bodyclose // linters bug
	if err != nil {
		return DatabaseEntries{}, fmt.Errorf(DoRequestError, err)
	}

	dbEntries := DatabaseEntries{}
	err = ReadAndUnmarshal(respSecretsList, &dbEntries)
	return dbEntries, err
}

func ReadAndUnmarshal(resp *http.Response, target any) error {
	var buf bytes.Buffer
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("failed to authenticate with the given credentials: %d %s", resp.StatusCode, buf.String())
	}
	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf.Bytes(), target)
}

func (api *API) ListDatabases() (Databases, error) {
	listDBRequest, err := http.NewRequest("GET", api.getendpointURL("list"), http.NoBody)
	if err != nil {
		return Databases{}, fmt.Errorf("error: creating db request: %w", err)
	}

	respDBList, err := api.doAuthenticatedRequest(listDBRequest) //nolint:bodyclose // linters bug
	if err != nil {
		return Databases{}, fmt.Errorf(DoRequestError, err)
	}

	databases := Databases{}
	err = ReadAndUnmarshal(respDBList, &databases)
	return databases, err
}

func (api *API) GetSecret(database, secretName string) (SecretEntry, error) {
	dbFingerprint, err := api.getDatabaseFingerprint(database)
	if err != nil {
		return SecretEntry{}, fmt.Errorf("error: getting DB fingerprint: %w", err)
	}

	secretFingerprint, err := api.getSecretFingerprint(dbFingerprint, secretName, "")
	if err != nil {
		return SecretEntry{}, fmt.Errorf("error: getting Secret fingerprint: %w", err)
	}
	readSecretRequest, err := http.NewRequest("GET", api.getendpointURL(fmt.Sprintf("read?db=%s&entry=%s", dbFingerprint, secretFingerprint)), http.NoBody)
	if err != nil {
		return SecretEntry{}, fmt.Errorf("error: creating secrets request: %w", err)
	}

	respSecretRead, err := api.doAuthenticatedRequest(readSecretRequest) //nolint:bodyclose // linters bug
	if err != nil {
		return SecretEntry{}, fmt.Errorf(DoRequestError, err)
	}

	secretEntry := SecretEntry{}
	err = ReadAndUnmarshal(respSecretRead, &secretEntry)
	return secretEntry, err
}

func (s SecretEntry) ToMap() map[string][]byte {
	m := make(map[string][]byte)

	m["name"] = []byte(s.Name)
	m["fingerprint"] = []byte(s.Fingerprint)
	m["itemclass"] = []byte(s.Itemclass)
	m["login"] = []byte(s.Login)
	m["pass"] = []byte(s.Pass)
	m["url"] = []byte(s.URL)
	m["importance"] = []byte(s.Importance)
	m["comment"] = []byte(s.Comment)
	m["expirydate"] = []byte(s.Expirydate)
	m["tags"] = []byte(s.Tags)
	m["author"] = []byte(s.Author)
	m["category"] = []byte(s.Category)
	m["icon"] = []byte(s.Icon)
	m["secondeye"] = []byte(s.Secondeye)
	m["secondpass"] = []byte(s.Secondpass)
	m["template"] = []byte(s.Template)
	m["acm"] = []byte(s.Acm)
	m["paramstr"] = []byte(s.Paramstr)
	m["loginid"] = []byte(s.Loginid)
	m["passid"] = []byte(s.Passid)
	m["donotaddon"] = []byte(s.Donotaddon)
	m["markassafe"] = []byte(s.Markassafe)
	m["safemode"] = []byte(s.Safemode)

	return m
}
