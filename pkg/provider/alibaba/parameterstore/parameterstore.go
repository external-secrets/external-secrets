package parameterstore

import (
	"context"
	"encoding/json"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	oossdk "github.com/alibabacloud-go/oos-20190601/v3/client"
	"github.com/avast/retry-go/v4"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/commonutil"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var _ esv1beta1.SecretsClient = &ParameterStore{}

type ParameterStore struct {
	Client PSInterface
	Config *openapi.Config
}

type PSInterface interface {
	GetSecretValue(ctx context.Context, request *oossdk.GetSecretParameterRequest) (*oossdk.GetSecretParameterResponseBody, error)
	Endpoint() string
}

func (ps *ParameterStore) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	//TODO implement me
	return fmt.Errorf("not implemented")
}

func (ps *ParameterStore) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	//TODO implement me
	return fmt.Errorf("not implemented")
}

func (ps *ParameterStore) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	//TODO implement me
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

func (ps *ParameterStore) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	//TODO implement me
	data, err := ps.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
	}
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

// GetSecret returns a single secret from the provider.
func (ps *ParameterStore) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(ps.Client) {
		return nil, fmt.Errorf(auth.ErrUninitializedAlibabaProvider)
	}

	request := &oossdk.GetSecretParameterRequest{
		Name:           &ref.Key,
		WithDecryption: &[]bool{true}[0],
	}

	if ref.Version != "" {
		intVersion, err := strconv.ParseInt(ref.Version, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid version")
		}
		intVersion32 := int32(intVersion)
		request.ParameterVersion = &intVersion32
	}

	secretOut, err := ps.Client.GetSecretValue(ctx, request)
	if err != nil {
		return nil, commonutil.SanitizeErr(err)
	}

	if ref.Property == "" {
		if secretOut.Parameter != nil {
			return []byte(utils.Deref(utils.Deref(secretOut.Parameter).Value)), nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string nor binary for key: %s", ref.Key)
	}
	var payload string
	if secretOut.Parameter != nil {
		payload = utils.Deref(utils.Deref(secretOut.Parameter).Value)
	}
	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// NewClient constructs a new secrets client based on the provided store.
func NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	credentials, err := auth.NewAuth(ctx, kube, store, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alibaba credentials: %w", err)
	}

	config := &openapi.Config{
		RegionId:   utils.Ptr(alibabaSpec.RegionID),
		Credential: credentials,
	}

	options := auth.NewOptions(store)
	client, err := newClient(config, options)
	if err != nil {
		return nil, fmt.Errorf(auth.ErrAlibabaClient, err)
	}

	ps := &ParameterStore{}

	ps.Client = client
	ps.Config = config
	return ps, nil
}

func (ps *ParameterStore) Close(_ context.Context) error {
	return nil
}

func (ps *ParameterStore) Validate() (esv1beta1.ValidationResult, error) {
	err := retry.Do(
		func() error {
			_, err := ps.Config.Credential.GetSecurityToken()
			if err != nil {
				return err
			}

			return nil
		},
		retry.Attempts(5),
	)
	if err != nil {
		return esv1beta1.ValidationResultError, commonutil.SanitizeErr(err)
	}

	return esv1beta1.ValidationResultReady, nil
}
