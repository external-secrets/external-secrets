/*
Copyright © The ESO Authors

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

package webhook

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

// KrbTransport is a wrapper around http.Transport that adds Kerberos authentication to requests.
type KrbTransport struct {
	*http.Transport
	krbConfig *config.Config
}

func parseUserString(userString string) (userParts []string, err error) {
	if strings.Contains(userString, "@") {
		userParts = strings.Split(userString, "@")
		if len(userParts) != 2 || userParts[0] == "" || userParts[1] == "" {
			return nil, errors.New("unable to parse user, expecting non-empty 'user@domain' format")
		}
	} else if strings.Contains(userString, "\\") {
		err := errors.New(`Unable to parse user and domain, expecting 'user@domain' format`)
		return nil, err
	} else {
		err := errors.New(`Unable to parse user, expecting 'user@domain' format`)
		return nil, err
	}

	return userParts, nil
}

// RoundTrip implements the http.RoundTripper interface for Kerberos authentication.
func (t *KrbTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	userString, pass, ok := req.BasicAuth()

	if !ok {
		return t.Transport.RoundTrip(req)
	}
	var err error

	// perform username, domain extraction
	userStringParsed, err := parseUserString(userString)
	if err != nil {
		return nil, err
	}
	user := strings.ReplaceAll(userStringParsed[0], "\n", "")
	domain := strings.ReplaceAll(userStringParsed[1], "\n", "") // some string sanitization

	if t.krbConfig == nil {
		t.krbConfig = config.New()
		t.krbConfig.LibDefaults.DNSLookupKDC = true // lookup KDC in DNS
	}

	cl := client.NewWithPassword(user, domain, pass, t.krbConfig, client.DisablePAFXFAST(true))
	err = cl.Login()
	if err != nil {
		return nil, err
	}
	h := req.URL.Hostname()
	err = spnego.SetSPNEGOHeader(cl, req, "HTTP/"+h)
	if err != nil {
		return nil, err
	}
	return t.Transport.RoundTrip(req)
}
