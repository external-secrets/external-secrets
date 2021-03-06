package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func TestProvider(t *testing.T) {
	cl := clientfake.NewClientBuilder().Build()
	p := Provider{}

	tbl := []struct {
		test   string
		store  esv1alpha1.GenericStore
		expErr bool
	}{
		{
			test:   "should not create provider due to nil store",
			store:  nil,
			expErr: true,
		},
		{
			test:   "should not create provider due to missing provider",
			expErr: true,
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{},
			},
		},
		{
			test:   "should not create provider due to missing provider field",
			expErr: true,
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{},
				},
			},
		},

		{
			test:   "should create provider",
			expErr: false,
			store: &esv1alpha1.SecretStore{
				Spec: esv1alpha1.SecretStoreSpec{
					Provider: &esv1alpha1.SecretStoreProvider{
						AWS: &esv1alpha1.AWSProvider{
							Service: esv1alpha1.AWSServiceParameterStore,
						},
					},
				},
			},
		},
	}
	for i := range tbl {
		row := tbl[i]
		t.Run(row.test, func(t *testing.T) {
			sc, err := p.NewClient(context.TODO(), row.store, cl, "foo")
			if row.expErr {
				assert.Error(t, err)
				assert.Nil(t, sc)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, sc)
			}
		})
	}
}
