package template

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	errCreatingConfig    = "error creating cluster configuration: %v"
	errCreatingClientset = "error creating Kubernetes clientset: %v"
	errFetchingSecret    = "error fetching secret: %v"
	errKeyNotFound       = "key %s not found in secret %s"
)

func getSecretKey(secretName, namespace, keyName string) (string, error) {
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return "", fmt.Errorf(errCreatingConfig, err)
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return "", fmt.Errorf(errCreatingClientset, err)
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf(errFetchingSecret, err)
	}
	val, ok := secret.Data[keyName]
	if !ok {
		return "", fmt.Errorf(errKeyNotFound, keyName, secretName)
	}
	return string(val), nil
}
