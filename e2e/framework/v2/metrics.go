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

package v2

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type MetricSample struct {
	Name   string
	Labels map[string]string
	Value  float64
}

type MetricsMap map[string][]MetricSample

func ScrapeControllerMetrics(ctx context.Context, config *rest.Config, clientset kubernetes.Interface, namespace string) (MetricsMap, error) {
	podName, err := findPod(ctx, clientset, namespace, "app.kubernetes.io/name=external-secrets")
	if err != nil {
		return nil, err
	}

	return scrapePodMetrics(ctx, config, clientset, namespace, podName, 8080)
}

func ScrapeProviderMetrics(ctx context.Context, config *rest.Config, clientset kubernetes.Interface, namespace, providerName string) (MetricsMap, error) {
	labelSelector := fmt.Sprintf("app.kubernetes.io/name=external-secrets-provider-%s", providerName)
	podName, err := findPod(ctx, clientset, namespace, labelSelector)
	if err != nil {
		return nil, err
	}

	return scrapePodMetrics(ctx, config, clientset, namespace, podName, 8081)
}

func GetMetricValue(metrics MetricsMap, metricName string, matchLabels map[string]string) (float64, bool) {
	samples, exists := metrics[metricName]
	if !exists {
		return 0, false
	}

	for _, sample := range samples {
		if labelsMatch(sample.Labels, matchLabels) {
			return sample.Value, true
		}
	}

	return 0, false
}

func ExpectMetricExists(metrics MetricsMap, metricName string) {
	_, exists := metrics[metricName]
	if !exists {
		availableMetrics := []string{}
		for name := range metrics {
			availableMetrics = append(availableMetrics, name)
		}
		GinkgoWriter.Printf("Available metrics: %v\n", availableMetrics)
	}
	Expect(exists).To(BeTrue(), "metric %s should exist", metricName)
}

func ExpectMetricValue(metrics MetricsMap, metricName string, matchLabels map[string]string, expectedValue float64) {
	value, found := GetMetricValue(metrics, metricName, matchLabels)
	Expect(found).To(BeTrue(), "metric %s with labels %v should exist", metricName, matchLabels)
	Expect(value).To(Equal(expectedValue), "metric %s value mismatch", metricName)
}

func ExpectMetricGreaterThan(metrics MetricsMap, metricName string, matchLabels map[string]string, threshold float64) {
	value, found := GetMetricValue(metrics, metricName, matchLabels)
	Expect(found).To(BeTrue(), "metric %s with labels %v should exist", metricName, matchLabels)
	Expect(value).To(BeNumerically(">", threshold), "metric %s should be greater than %f", metricName, threshold)
}

func WaitForMetric(ctx context.Context, scraper func() (MetricsMap, error), metricName string, matchLabels map[string]string, minValue float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		metrics, err := scraper()
		if err == nil {
			value, found := GetMetricValue(metrics, metricName, matchLabels)
			if found && value >= minValue {
				return nil
			}
		}
		time.Sleep(time.Second)
	}

	return fmt.Errorf("timeout waiting for metric %s with labels %v to reach %f", metricName, matchLabels, minValue)
}

func scrapePodMetrics(ctx context.Context, config *rest.Config, clientset kubernetes.Interface, namespace, podName string, podPort int) (MetricsMap, error) {
	address, cleanup, err := setupPortForward(ctx, config, clientset, namespace, podName, podPort)
	if err != nil {
		return nil, fmt.Errorf("failed to setup port forward: %w", err)
	}
	defer cleanup()

	time.Sleep(time.Second)

	body, err := scrapeMetrics(ctx, address)
	if err != nil {
		return nil, err
	}

	return parsePrometheusMetrics(body), nil
}

func findPod(ctx context.Context, clientset kubernetes.Interface, namespace, labelSelector string) (string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}

	return "", fmt.Errorf("no running pod found for selector %s", labelSelector)
}

func setupPortForward(ctx context.Context, config *rest.Config, clientset kubernetes.Interface, namespace, podName string, podPort int) (string, func(), error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("failed to get pod: %w", err)
	}
	if pod.Status.Phase != corev1.PodRunning {
		return "", nil, fmt.Errorf("pod %s is not running: %s", podName, pod.Status.Phase)
	}

	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create round tripper: %w", err)
	}

	url := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward").
		URL()

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{}, 1)
	ports := []string{fmt.Sprintf("0:%d", podPort)}

	pf, err := portforward.New(dialer, ports, stopChan, readyChan, GinkgoWriter, GinkgoWriter)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	errChan := make(chan error, 1)
	go func() {
		if forwardErr := pf.ForwardPorts(); forwardErr != nil {
			errChan <- forwardErr
		}
	}()

	select {
	case <-readyChan:
		forwardedPorts, portErr := pf.GetPorts()
		if portErr != nil {
			close(stopChan)
			return "", nil, fmt.Errorf("failed to get forwarded ports: %w", portErr)
		}
		if len(forwardedPorts) == 0 {
			close(stopChan)
			return "", nil, fmt.Errorf("no ports were forwarded")
		}
		cleanup := func() {
			close(stopChan)
		}
		return fmt.Sprintf("localhost:%d", forwardedPorts[0].Local), cleanup, nil
	case err = <-errChan:
		close(stopChan)
		return "", nil, fmt.Errorf("port forward failed: %w", err)
	case <-time.After(30 * time.Second):
		close(stopChan)
		return "", nil, fmt.Errorf("timeout waiting for port forward to be ready")
	}
}

func scrapeMetrics(ctx context.Context, address string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s/metrics", address), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to scrape metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(body), nil
}

func parsePrometheusMetrics(body string) MetricsMap {
	metrics := make(MetricsMap)
	metricRegex := regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*?)(?:\{([^}]*)\})?\s+([^\s]+)`)

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		matches := metricRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}

		value, err := strconv.ParseFloat(matches[3], 64)
		if err != nil {
			continue
		}

		sample := MetricSample{
			Name:   matches[1],
			Labels: parseLabels(matches[2]),
			Value:  value,
		}
		metrics[sample.Name] = append(metrics[sample.Name], sample)
	}

	return metrics
}

func parseLabels(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	labelRegex := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)="([^"]*)"`)
	matches := labelRegex.FindAllStringSubmatch(labelsStr, -1)
	for _, match := range matches {
		if len(match) == 3 {
			labels[match[1]] = match[2]
		}
	}
	return labels
}

func labelsMatch(sampleLabels, matchLabels map[string]string) bool {
	for key, value := range matchLabels {
		if sampleLabels[key] != value {
			return false
		}
	}
	return true
}
