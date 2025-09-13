/*
Copyright © 2025 ESO Maintainer Team

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
package addon

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/external-secrets/external-secrets-e2e/framework/log"
	. "github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForward struct {
	kubeClient       kubernetes.Interface
	restcfg          *rest.Config
	serviceName      string
	serviceNamespace string
	containerPort    int

	localPort int
	fwd       *portforward.PortForwarder
}

func NewPortForward(cl kubernetes.Interface, restcfg *rest.Config, serviceName, serviceNamespace string, containerPort int) (*PortForward, error) {
	pf := &PortForward{
		kubeClient:       cl,
		restcfg:          restcfg,
		serviceName:      serviceName,
		serviceNamespace: serviceNamespace,
		containerPort:    containerPort,
	}

	return pf, nil
}

// setupPortForward creates port-forward connections to the vault service
func (pf *PortForward) Start() error {
	localPort, err := findAvailablePort()
	if err != nil {
		return fmt.Errorf("unable to find available port: %w", err)
	}
	pf.localPort = localPort

	svc, err := pf.kubeClient.CoreV1().Services(pf.serviceNamespace).Get(GinkgoT().Context(), pf.serviceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get service %s: %w", pf.serviceName, err)
	}

	selector := metav1.LabelSelector{MatchLabels: svc.Spec.Selector}
	pods, err := pf.kubeClient.CoreV1().Pods(pf.serviceNamespace).List(GinkgoT().Context(), metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&selector),
	})
	if err != nil || len(pods.Items) == 0 {
		return fmt.Errorf("unable to find pods for service %s: %w", pf.serviceName, err)
	}

	pod := pods.Items[0]

	// Create port-forward request
	req := pf.kubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(pf.restcfg)
	if err != nil {
		return fmt.Errorf("unable to create transport: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	ports := []string{fmt.Sprintf("%d:%d", pf.localPort, pf.containerPort)}

	var fwd *portforward.PortForwarder
	var stdout, stderr bytes.Buffer
	stopChan := make(chan struct{})
	readyChan := make(chan struct{})
	fwd, err = portforward.New(dialer, ports, stopChan, readyChan, &stdout, &stderr)
	if err != nil {
		return fmt.Errorf("unable to create port-forward: %w", err)
	}
	pf.fwd = fwd

	// run ForwardPorts in the background and capture the error (if any)
	errChan := make(chan error, 1)
	go func() {
		if err := fwd.ForwardPorts(); err != nil {
			log.Logf("port-forward error: %v %s %s", err, stdout.String(), stderr.String())
			errChan <- err
			return
		}
		// signal normal termination (e.g., Close called later)
		errChan <- nil
	}()

	// avoid indefinite wait if forwarder fails before signaling ready
	ctx := GinkgoT().Context()
	select {
	case <-readyChan:
		return nil
	case err := <-errChan:
		if err == nil {
			return fmt.Errorf("port-forward terminated before readiness without error: %s %s", stdout.String(), stderr.String())
		}
		return fmt.Errorf("unable to start port-forward: %w: %s %s", err, stdout.String(), stderr.String())
	case <-time.After(10 * time.Second):
		close(stopChan)
		return fmt.Errorf("timeout waiting for port-forward readiness: %s %s", stdout.String(), stderr.String())
	case <-ctx.Done():
		close(stopChan)
		return fmt.Errorf("context canceled waiting for port-forward readiness: %w", ctx.Err())
	}
}

func (pf *PortForward) Close() {
	pf.fwd.Close()
}

// findAvailablePort finds an available local port
func findAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
