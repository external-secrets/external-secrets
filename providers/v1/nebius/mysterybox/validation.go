// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package mysterybox

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"golang.org/x/net/idna"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	errNilStore                                = "found nil store"
	errMissingStoreSpec                        = "store is missing spec"
	errMissingProvider                         = "storeSpec is missing provider"
	errInvalidProvider                         = "invalid provider spec. Missing nebiusmysterybox field in store %s"
	errMissingAuthOptions                      = "invalid auth configuration: none provided"
	errInvalidAuthConfig                       = "invalid auth configuration: exactly one must be specified"
	errInvalidTokenAuthConfig                  = "invalid token auth configuration: no secret key specified"
	errInvalidSACredsAuthConfig                = "invalid ServiceAccount creds auth configuration: no secret key specified"
	errFailedToRetrieveToken                   = "failed to retrieve iam token by credentials: %w"
	errMissingAPIDomain                        = "API domain must be set"
	errInvalidAPIDomain                        = "API domain is not valid"
	errInvalidAPIDomainWithError               = errInvalidAPIDomain + ": %w"
	errInvalidCertificateConfigNoNameSpecified = "invalid certificate configuration: no name specified"
	errInvalidCertificateConfigNoKeySpecified  = "invalid certificate configuration: no key specified"
)

var labelRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
var numericRe = regexp.MustCompile(`^\d+$`)

// ValidateStore validates the given Store and its associated properties for correctness and configuration integrity.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	provider, err := getNebiusMysteryboxProvider(store)
	if err != nil {
		return nil, err
	}

	err = p.validateAPIDomain(provider.APIDomain)
	if err != nil {
		return nil, err
	}

	if err := validateProviderAuth(provider); err != nil {
		return nil, err
	}

	var selectors []*esmeta.SecretKeySelector

	if provider.Auth.Token.Name != "" {
		selectors = append(selectors, &provider.Auth.Token)
	}
	if provider.Auth.ServiceAccountCreds.Name != "" {
		selectors = append(selectors, &provider.Auth.ServiceAccountCreds)
	}
	if provider.CAProvider != nil {
		err := validateCertificate(&provider.CAProvider.Certificate)
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, &provider.CAProvider.Certificate)
	}

	for _, selector := range selectors {
		if err := esutils.ValidateSecretSelector(store, *selector); err != nil {
			return nil, err
		}
		if err := esutils.ValidateReferentSecretSelector(store, *selector); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func getNebiusMysteryboxProvider(store esv1.GenericStore) (*esv1.NebiusMysteryboxProvider, error) {
	if store == nil {
		return nil, errors.New(errNilStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, errors.New(errMissingStoreSpec)
	}
	if spc.Provider == nil {
		return nil, errors.New(errMissingProvider)
	}
	provider := spc.Provider.NebiusMysterybox
	if provider == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetNamespacedName())
	}
	return provider, nil
}

func validateProviderAuth(provider *esv1.NebiusMysteryboxProvider) error {
	if provider.Auth.Token.Name == "" && provider.Auth.ServiceAccountCreds.Name == "" {
		return errors.New(errMissingAuthOptions)
	}
	if provider.Auth.Token.Name != "" && provider.Auth.ServiceAccountCreds.Name != "" {
		return errors.New(errInvalidAuthConfig)
	}
	if provider.Auth.Token.Name != "" && provider.Auth.Token.Key == "" {
		return errors.New(errInvalidTokenAuthConfig)
	}
	if provider.Auth.ServiceAccountCreds.Name != "" && provider.Auth.ServiceAccountCreds.Key == "" {
		return errors.New(errInvalidSACredsAuthConfig)
	}
	return nil
}

func (p *Provider) validateAPIDomain(apiDomain string) error {
	if strings.TrimSpace(apiDomain) == "" {
		return errors.New(errMissingAPIDomain)
	}
	if strings.Contains(apiDomain, "://") {
		return fmt.Errorf(errInvalidAPIDomainWithError, errors.New("scheme is not allowed"))
	}

	var err error
	host := apiDomain
	// API Domain is allowed without port, it will be added in sdk in case it's missed
	if strings.Contains(apiDomain, ":") {
		host, _, err = net.SplitHostPort(apiDomain)
		if err != nil {
			p.Logger.Info("Error parsing apiDomain", "err", err)
			return fmt.Errorf(errInvalidAPIDomainWithError, err)
		}
	}

	err = validateDomainByRFC(host)
	if err != nil {
		p.Logger.Info("Error validating apiDomain", "err", err)
		return fmt.Errorf(errInvalidAPIDomainWithError, err)
	}

	return nil
}

func validateDomainByRFC(domain string) error {
	if strings.HasSuffix(domain, ".") {
		return errors.New("domain must not end with a dot")
	}
	ascii, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return err
	}
	if ascii == "" || len(ascii) > 253 {
		return errors.New("domain length out of range")
	}
	labels := strings.Split(ascii, ".")
	if len(labels) < 2 {
		return errors.New("domain must contain at least one dot")
	}
	for _, l := range labels {
		if len(l) < 1 || len(l) > 63 {
			return errors.New("label length out of range")
		}
		if !labelRe.MatchString(l) {
			return errors.New("invalid label: " + l)
		}
	}
	tld := labels[len(labels)-1]
	if numericRe.MatchString(tld) {
		return errors.New("numeric TLD is not allowed")
	}
	return nil
}

func validateCertificate(certificate *esmeta.SecretKeySelector) error {
	if certificate.Name == "" {
		return errors.New(errInvalidCertificateConfigNoNameSpecified)
	}
	if certificate.Key == "" {
		return errors.New(errInvalidCertificateConfigNoKeySpecified)
	}
	return nil
}
