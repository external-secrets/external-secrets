/*
Copyright 2020 The cert-manager Authors.
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

package addon

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type HelmServer struct {
	ChartDir      string
	ChartRevision string

	config   *Config
	srv      *http.Server
	serveDir string
}

func (s *HelmServer) Setup(config *Config) error {
	s.config = config
	var err error
	s.serveDir, err = os.MkdirTemp("", "e2e-helm")
	if err != nil {
		return err
	}

	// nolint:gosec
	cmd := exec.Command("helm", "package", s.ChartDir, "--version", s.ChartRevision)
	cmd.Dir = s.serveDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to package helm chart: %w %s", err, string(out))
	}

	cmd = exec.Command("helm", "repo", "index", ".")
	cmd.Dir = s.serveDir
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to create helm index: %w %s", err, string(out))
	}

	_, err = s.config.KubeClientSet.CoreV1().Services("default").Create(context.Background(), &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "e2e-helmserver",
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				// set via e2e/run.sh
				"app": "eso-e2e",
			},
			Ports: []v1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(3000),
				},
			}},
	}, metav1.CreateOptions{})
	return err
}

func (s *HelmServer) Install() error {
	fs := http.FileServer(http.Dir(s.serveDir))
	http.Handle("/", fs)
	s.srv = &http.Server{Addr: ":3000"}
	go func() {
		_ = s.srv.ListenAndServe()
	}()
	return nil
}

func (s *HelmServer) Logs() error {
	return nil
}

func (s *HelmServer) Uninstall() error {
	err := s.config.KubeClientSet.CoreV1().Services("default").Delete(context.Background(), "e2e-helmserver", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return s.srv.Close()
}
