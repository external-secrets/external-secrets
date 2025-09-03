package template

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/client-go/kubernetes"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Error constants for Kubernetes operations
var (
	errCreatingConfig    = errors.New("error creating cluster configuration")
	errCreatingClientset = errors.New("error creating Kubernetes clientset")
	errFetchingSecret    = errors.New("error fetching secret")
	errKeyNotFound       = errors.New("key not found in secret")
)

func getSecretKey(secretName, namespace, keyName string) (string, error) {
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return "", fmt.Errorf("%w: %v", errCreatingConfig, err)
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errCreatingClientset, err)
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("%w: %v", errFetchingSecret, err)
	}
	val, ok := secret.Data[keyName]
	if !ok {
		return "", fmt.Errorf("%w: %q", errKeyNotFound, keyName)
	}
	return string(val), nil
}
