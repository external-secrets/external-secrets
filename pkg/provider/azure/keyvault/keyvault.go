package keyvault

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"golang.org/x/crypto/pkcs12"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

type Azure struct {
	kube       client.Client
	store      esv1alpha1.GenericStore
	baseClient *keyvault.BaseClient
	namespace  string
	iAzure     IAzure
}

type IAzure interface {
	getKeyVaultSecrets(ctx context.Context, vaultName string, version string, secretName string, withTags bool) (map[string][]byte, error)
}

func init() {
	schema.Register(&Azure{}, &esv1alpha1.SecretStoreProvider{
		AzureKV: &esv1alpha1.AzureKVProvider{},
	})
}

func (a *Azure) New(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.Provider, error) {
	anAzure := &Azure{
		kube:      kube,
		store:     store,
		namespace: namespace,
	}
	anAzure.iAzure = anAzure
	azClient, err := anAzure.newAzureClient(ctx)

	if err != nil {
		return nil, err
	}

	anAzure.baseClient = azClient
	return anAzure, nil
}

// implement store.Client.GetSecret Interface.
// retrieve a secret with the secret name defined in ref.Property in a specific keyvault with the name ref.Name.
func (a *Azure) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	version := ""
	var secretBundle []byte

	if ref.Version != "" {
		version = ref.Version
	}
	secretName := ref.Property
	nameSplitted := strings.Split(secretName, "_")
	getTags := false

	if nameSplitted[len(nameSplitted)-1] == "TAG" {
		secretName = nameSplitted[0]
		getTags = true
	}

	secretMap, err := a.iAzure.getKeyVaultSecrets(ctx, ref.Key, version, secretName, getTags)
	if err != nil {
		return nil, err
	}

	secretBundle = secretMap[ref.Property]
	return secretBundle, nil
}

// implement store.Client.GetSecretMap Interface.
// retrieve ALL secrets in a specific keyvault with the name ref.Name.
func (a *Azure) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secretMap, err := a.iAzure.getKeyVaultSecrets(ctx, ref.Key, ref.Version, "", true)
	return secretMap, err
}

// getCertBundle returns the certificate bundle.
func getCertBundleForPKCS(certificateRawVal string, certBundleOnly, certKeyOnly bool) (bundle string, err error) {
	pfx, err := base64.StdEncoding.DecodeString(certificateRawVal)

	if err != nil {
		return bundle, err
	}
	blocks, _ := pkcs12.ToPEM(pfx, "")

	for _, block := range blocks {
		// skip the private key if looking for the cert only
		if block.Type == "PRIVATE KEY" && certBundleOnly {
			continue
		}
		// no headers
		if block.Type == "PRIVATE KEY" {
			pkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				panic(err)
			}
			derStream := x509.MarshalPKCS1PrivateKey(pkey)
			block = &pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: derStream,
			}
			if certKeyOnly {
				bundle = string(pem.EncodeToMemory(block))
				break
			}
		}

		block.Headers = nil
		bundle += string(pem.EncodeToMemory(block))
	}
	return bundle, nil
}

// consolidated method to retrieve secret value or secrets list based on whether or not a secret name is passed.
// if the secret is of type PKCS then this is a cerificate that needs some decoding.
func (a *Azure) getKeyVaultSecrets(ctx context.Context, vaultName, version, secretName string, withTags bool) (map[string][]byte, error) {
	basicClient := a.baseClient
	secretsMap := make(map[string][]byte)
	certBundleOnly := false
	certKeyOnly := false
	secretNameinBE := secretName

	if secretName != "" {
		nameSplitted := strings.Split(secretName, "_")
		if nameSplitted[len(nameSplitted)-1] == "CRT" {
			secretNameinBE = nameSplitted[0]
			certBundleOnly = true
		}
		if nameSplitted[len(nameSplitted)-1] == "KEY" {
			secretNameinBE = nameSplitted[0]
			certKeyOnly = true
		}

		secretResp, err := basicClient.GetSecret(context.Background(), "https://"+vaultName+".vault.azure.net", secretNameinBE, version)
		if err != nil {
			return nil, err
		}
		secretValue := *secretResp.Value

		// Azure currently supports only PKCS#12 or PEM, PEM will be taken as it is, PKCS needs processing
		if secretResp.ContentType != nil && *secretResp.ContentType == "application/x-pkcs12" {
			secretValue, err = getCertBundleForPKCS(*secretResp.Value, certBundleOnly, certKeyOnly)
			if err != nil {
				return nil, err
			}
		}
		secretsMap[secretName] = []byte(secretValue)
		if withTags {
			appendTagsToSecretMap(secretName, secretsMap, secretResp.Tags)
		}
	} else {
		secretList, err := basicClient.GetSecrets(context.Background(), "https://"+vaultName+".vault.azure.net", nil)
		if err != nil {
			return nil, err
		}
		for _, secret := range secretList.Values() {
			if !*secret.Attributes.Enabled {
				continue
			}
			secretResp, err := basicClient.GetSecret(context.Background(), "https://"+vaultName+".vault.azure.net", path.Base(*secret.ID), "")
			secretValue := *secretResp.Value
			// Azure currently supports only PKCS#12 or PEM, PEM will be taken as it is, PKCS needs processing
			if secretResp.ContentType != nil && *secretResp.ContentType == "application/x-pkcs12" {
				secretValue, err = getCertBundleForPKCS(*secretResp.Value, certBundleOnly, certKeyOnly)
			}
			if err != nil {
				return nil, err
			}
			secretsMap[path.Base(*secret.ID)] = []byte(secretValue)
			if withTags {
				appendTagsToSecretMap(path.Base(*secret.ID), secretsMap, secretResp.Tags)
			}
		}
	}
	return secretsMap, nil
}

func appendTagsToSecretMap(secretName string, secretsMap map[string][]byte, tags map[string]*string) {
	for tagKey, tagValue := range tags {
		secretsMap[secretName+"_"+tagKey+"_TAG"] = []byte(*tagValue)
	}
}
func (a *Azure) newAzureClient(ctx context.Context) (*keyvault.BaseClient, error) {
	spec := *a.store.GetSpec().Provider.AzureKV
	tenantID := *spec.TenantID

	if spec.AuthSecretRef == nil {
		return nil, fmt.Errorf("missing clientID/clientSecret in store config")
	}
	scoped := true
	if a.store.GetObjectMeta().String() == "ClusterSecretStore" {
		scoped = false
	}
	if spec.AuthSecretRef.ClientID == nil || spec.AuthSecretRef.ClientSecret == nil {
		return nil, fmt.Errorf("missing accessKeyID/secretAccessKey in store config")
	}
	cid, err := a.secretKeyRef(ctx, a.store.GetNamespacedName(), *spec.AuthSecretRef.ClientID, scoped)
	if err != nil {
		return nil, err
	}
	csec, err := a.secretKeyRef(ctx, a.store.GetNamespacedName(), *spec.AuthSecretRef.ClientSecret, scoped)
	if err != nil {
		return nil, err
	}
	os.Setenv("AZURE_TENANT_ID", tenantID)
	os.Setenv("AZURE_CLIENT_ID", cid)
	os.Setenv("AZURE_CLIENT_SECRET", csec)

	authorizer, err := kvauth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}
	os.Unsetenv("AZURE_TENANT_ID")
	os.Unsetenv("AZURE_CLIENT_ID")
	os.Unsetenv("AZURE_CLIENT_SECRET")

	basicClient := keyvault.New()
	basicClient.Authorizer = authorizer

	return &basicClient, nil
}
func (a *Azure) secretKeyRef(ctx context.Context, namespace string, secretRef smmeta.SecretKeySelector, scoped bool) (string, error) {
	var secret corev1.Secret
	ref := types.NamespacedName{
		Namespace: namespace,
		Name:      secretRef.Name,
	}
	if !scoped && secretRef.Namespace != nil {
		ref.Namespace = *secretRef.Namespace
	}
	err := a.kube.Get(ctx, ref, &secret)
	if err != nil {
		return "", err
	}
	keyBytes, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf("no data for %q in secret '%s/%s'", secretRef.Key, secretRef.Name, namespace)
	}
	value := strings.TrimSpace(string(keyBytes))
	return value, nil
}
