//Copyright External Secrets Inc. All Rights Reserved

package secretmanager

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

func NewTokenSource(ctx context.Context, auth esv1beta1.GCPSMAuth, projectID, storeKind string, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	ts, err := serviceAccountTokenSource(ctx, auth, storeKind, kube, namespace)
	if ts != nil || err != nil {
		return ts, err
	}
	wi, err := newWorkloadIdentity(ctx, projectID)
	if err != nil {
		return nil, errors.New("unable to initialize workload identity")
	}
	defer wi.Close()
	isClusterKind := storeKind == esv1beta1.ClusterSecretStoreKind
	ts, err = wi.TokenSource(ctx, auth, isClusterKind, kube, namespace)
	if ts != nil || err != nil {
		return ts, err
	}
	return google.DefaultTokenSource(ctx, CloudPlatformRole)
}

func serviceAccountTokenSource(ctx context.Context, auth esv1beta1.GCPSMAuth, storeKind string, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	sr := auth.SecretRef
	if sr == nil {
		return nil, nil
	}
	credentials, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		namespace,
		&auth.SecretRef.SecretAccessKey)
	if err != nil {
		return nil, err
	}
	config, err := google.JWTConfigFromJSON([]byte(credentials), CloudPlatformRole)
	if err != nil {
		return nil, fmt.Errorf(errUnableProcessJSONCredentials, err)
	}
	return config.TokenSource(ctx), nil
}
