/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MetricSample represents a single Prometheus metric sample
type MetricSample struct {
	Name   string
	Labels map[string]string
	Value  float64
}

// MetricsMap is a map of metric names to their samples
type MetricsMap map[string][]MetricSample

// setupPortForward creates a port-forward to a pod and returns the local address and cleanup function
func setupPortForward(ctx context.Context, config *rest.Config, clientset *kubernetes.Clientset, namespace, podName string, podPort int) (string, func(), error) {
	// Find an available local port
	localPort := 0 // Let the system choose

	// Get the pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("failed to get pod: %w", err)
	}

	// Ensure pod is running
	if pod.Status.Phase != v1.PodRunning {
		return "", nil, fmt.Errorf("pod %s is not running: %s", podName, pod.Status.Phase)
	}

	// Create the port-forward request
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create round tripper: %w", err)
	}

	// Build the URL for port forwarding
	url := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward").
		URL()

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, url)

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{}, 1)

	// Create port forwarder
	ports := []string{fmt.Sprintf("%d:%d", localPort, podPort)}
	
	out := GinkgoWriter
	errOut := GinkgoWriter

	pf, err := portforward.New(dialer, ports, stopChan, readyChan, out, errOut)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	// Start port forwarding in background
	errChan := make(chan error, 1)
	go func() {
		if err := pf.ForwardPorts(); err != nil {
			errChan <- err
		}
	}()

	// Wait for ready or error
	select {
	case <-readyChan:
		// Port forward is ready
		forwardedPorts, err := pf.GetPorts()
		if err != nil {
			close(stopChan)
			return "", nil, fmt.Errorf("failed to get forwarded ports: %w", err)
		}
		
		if len(forwardedPorts) == 0 {
			close(stopChan)
			return "", nil, fmt.Errorf("no ports were forwarded")
		}

		localAddr := fmt.Sprintf("localhost:%d", forwardedPorts[0].Local)
		
		cleanup := func() {
			close(stopChan)
		}

		return localAddr, cleanup, nil
	case err := <-errChan:
		close(stopChan)
		return "", nil, fmt.Errorf("port forward failed: %w", err)
	case <-time.After(30 * time.Second):
		close(stopChan)
		return "", nil, fmt.Errorf("timeout waiting for port forward to be ready")
	}
}

// scrapeMetrics fetches metrics from an HTTP endpoint
func scrapeMetrics(ctx context.Context, address string) (string, error) {
	url := fmt.Sprintf("http://%s/metrics", address)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
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

// parsePrometheusMetrics parses Prometheus text format into a structured map
func parsePrometheusMetrics(body string) MetricsMap {
	metrics := make(MetricsMap)
	
	// Regular expression to parse metric lines
	// Format: metric_name{label1="value1",label2="value2"} value
	// or: metric_name value
	metricRegex := regexp.MustCompile(`^([a-zA-Z_:][a-zA-Z0-9_:]*?)(?:\{([^}]*)\})?\s+([^\s]+)`)

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		matches := metricRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}

		name := matches[1]
		labelsStr := matches[2]
		valueStr := matches[3]

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		labels := parseLabels(labelsStr)

		sample := MetricSample{
			Name:   name,
			Labels: labels,
			Value:  value,
		}

		metrics[name] = append(metrics[name], sample)
	}

	return metrics
}

// parseLabels parses label string into a map
func parseLabels(labelsStr string) map[string]string {
	labels := make(map[string]string)
	
	if labelsStr == "" {
		return labels
	}

	// Split by comma, but respect quotes
	labelRegex := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)="([^"]*)"`)
	matches := labelRegex.FindAllStringSubmatch(labelsStr, -1)
	
	for _, match := range matches {
		if len(match) == 3 {
			labels[match[1]] = match[2]
		}
	}

	return labels
}

// getMetricValue finds a metric with specific labels and returns its value
func getMetricValue(metrics MetricsMap, metricName string, matchLabels map[string]string) (float64, bool) {
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

// labelsMatch checks if sample labels match the required labels
func labelsMatch(sampleLabels, matchLabels map[string]string) bool {
	for key, value := range matchLabels {
		if sampleLabels[key] != value {
			return false
		}
	}
	return true
}

// findControllerPod finds the external-secrets controller pod
func findControllerPod(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=external-secrets",
	})
	if err != nil {
		return "", fmt.Errorf("failed to list controller pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no controller pods found")
	}

	// Return the first running pod
	for _, pod := range pods.Items {
		if pod.Status.Phase == v1.PodRunning {
			return pod.Name, nil
		}
	}

	return "", fmt.Errorf("no running controller pods found")
}

// findProviderPod finds a provider pod by label
func findProviderPod(ctx context.Context, clientset *kubernetes.Clientset, namespace string, providerType string) (string, error) {
	labelSelector := fmt.Sprintf("app.kubernetes.io/name=external-secrets-provider-%s", providerType)
	
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list provider pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no %s provider pods found", providerType)
	}

	// Return the first running pod
	for _, pod := range pods.Items {
		if pod.Status.Phase == v1.PodRunning {
			return pod.Name, nil
		}
	}

	return "", fmt.Errorf("no running %s provider pods found", providerType)
}

// scrapeControllerMetrics scrapes metrics from the controller pod
func scrapeControllerMetrics(ctx context.Context, config *rest.Config, clientset *kubernetes.Clientset, namespace string) (MetricsMap, error) {
	podName, err := findControllerPod(ctx, clientset, namespace)
	if err != nil {
		return nil, err
	}

	address, cleanup, err := setupPortForward(ctx, config, clientset, namespace, podName, 8080)
	if err != nil {
		return nil, fmt.Errorf("failed to setup port forward: %w", err)
	}
	defer cleanup()

	// Give port-forward a moment to stabilize
	time.Sleep(1 * time.Second)

	body, err := scrapeMetrics(ctx, address)
	if err != nil {
		return nil, err
	}

	return parsePrometheusMetrics(body), nil
}

// scrapeProviderMetrics scrapes metrics from a provider pod
func scrapeProviderMetrics(ctx context.Context, config *rest.Config, clientset *kubernetes.Clientset, namespace string, providerType string) (MetricsMap, error) {
	podName, err := findProviderPod(ctx, clientset, namespace, providerType)
	if err != nil {
		return nil, err
	}

	address, cleanup, err := setupPortForward(ctx, config, clientset, namespace, podName, 8081)
	if err != nil {
		return nil, fmt.Errorf("failed to setup port forward: %w", err)
	}
	defer cleanup()

	// Give port-forward a moment to stabilize
	time.Sleep(1 * time.Second)

	body, err := scrapeMetrics(ctx, address)
	if err != nil {
		return nil, err
	}

	return parsePrometheusMetrics(body), nil
}

// waitForMetric polls until a metric reaches a minimum value or times out
func waitForMetric(ctx context.Context, scraper func() (MetricsMap, error), metricName string, matchLabels map[string]string, minValue float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		metrics, err := scraper()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		value, found := getMetricValue(metrics, metricName, matchLabels)
		if found && value >= minValue {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for metric %s with labels %v to reach %f", metricName, matchLabels, minValue)
}

// ExpectMetricExists asserts that a metric exists
func ExpectMetricExists(metrics MetricsMap, metricName string) {
	_, exists := metrics[metricName]
	if !exists {
		// Debug: print available metrics for troubleshooting
		availableMetrics := []string{}
		for name := range metrics {
			availableMetrics = append(availableMetrics, name)
		}
		GinkgoWriter.Printf("Available metrics: %v\n", availableMetrics)
	}
	Expect(exists).To(BeTrue(), "metric %s should exist", metricName)
}

// ExpectMetricValue asserts that a metric has a specific value with given labels
func ExpectMetricValue(metrics MetricsMap, metricName string, matchLabels map[string]string, expectedValue float64) {
	value, found := getMetricValue(metrics, metricName, matchLabels)
	Expect(found).To(BeTrue(), "metric %s with labels %v should exist", metricName, matchLabels)
	Expect(value).To(Equal(expectedValue), "metric %s value mismatch", metricName)
}

// ExpectMetricGreaterThan asserts that a metric value is greater than a threshold
func ExpectMetricGreaterThan(metrics MetricsMap, metricName string, matchLabels map[string]string, threshold float64) {
	value, found := getMetricValue(metrics, metricName, matchLabels)
	Expect(found).To(BeTrue(), "metric %s with labels %v should exist", metricName, matchLabels)
	Expect(value).To(BeNumerically(">", threshold), "metric %s should be greater than %f", metricName, threshold)
}

