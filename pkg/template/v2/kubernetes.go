package template

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetSecretValue(secretName, namespace, keyName string) ([]byte, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("erro criando config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("erro criando clientset: %w", err)
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar secret: %w", err)
	}
	val, ok := secret.Data[keyName]
	if !ok {
		return nil, fmt.Errorf("campo %s não encontrado na secret", keyName)
	}
	return val, nil
}
