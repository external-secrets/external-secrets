package addon

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework/log"
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

	svc, err := pf.kubeClient.CoreV1().Services(pf.serviceNamespace).Get(context.Background(), pf.serviceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get service %s: %w", pf.serviceName, err)
	}

	selector := metav1.LabelSelector{MatchLabels: svc.Spec.Selector}
	pods, err := pf.kubeClient.CoreV1().Pods(pf.serviceNamespace).List(context.Background(), metav1.ListOptions{
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
	fwd, err = portforward.New(dialer, ports, make(chan struct{}), make(chan struct{}), &stdout, &stderr)
	if err != nil {
		return fmt.Errorf("unable to create port-forward: %w", err)
	}
	pf.fwd = fwd

	go func() {
		if err := fwd.ForwardPorts(); err != nil {
			log.Logf("port-forward error: %v %s %s", err, stdout.String(), stderr.String())
		}
	}()

	// Wait a bit for port-forward to establish
	time.Sleep(2 * time.Second)

	return nil
}

func (pf *PortForward) Close() {
	if pf.fwd != nil {
		pf.fwd.Close()
	}
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
