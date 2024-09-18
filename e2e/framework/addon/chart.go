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
	"bytes"
	"context"
	"fmt"
	"os/exec"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework/log"
)

// HelmChart installs the specified Chart into the cluster.
type HelmChart struct {
	Namespace    string
	ReleaseName  string
	Chart        string
	ChartVersion string
	Repo         ChartRepo
	Vars         []StringTuple
	Values       []string
	Args         []string

	config *Config
}

type ChartRepo struct {
	Name string
	URL  string
}

type StringTuple struct {
	Key   string
	Value string
}

// Setup stores the config in an internal field
// to get access to the k8s api in orderto fetch logs.
func (c *HelmChart) Setup(cfg *Config) error {
	c.config = cfg
	return nil
}

// Install adds the chart repo and installs the helm chart.
func (c *HelmChart) Install() error {
	err := c.addRepo()
	if err != nil {
		return err
	}

	args := []string{"install", c.ReleaseName, c.Chart,
		"--dependency-update",
		"--debug",
		"--wait",
		"--timeout", "600s",
		"-o", "yaml",
		"--namespace", c.Namespace,
	}

	if c.ChartVersion != "" {
		args = append(args, "--version", c.ChartVersion)
	}

	for _, v := range c.Values {
		args = append(args, "--values", v)
	}

	for _, s := range c.Vars {
		args = append(args, "--set", fmt.Sprintf("%s=%s", s.Key, s.Value))
	}

	args = append(args, c.Args...)

	var sout, serr bytes.Buffer
	log.Logf("installing chart with args: %+q", args)
	cmd := exec.Command("helm", args...)
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("unable to run cmd: %w: %s %s", err, sout.String(), serr.String())
	}
	return nil
}

// Uninstall removes the chart aswell as the repo.
func (c *HelmChart) Uninstall() error {
	var sout, serr bytes.Buffer
	args := []string{"delete", "--namespace", c.Namespace, c.ReleaseName}
	cmd := exec.Command("helm", args...)
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("unable to delete helm release: %w: %s, %s", err, sout.String(), serr.String())
	}
	return c.removeRepo()
}

func (c *HelmChart) addRepo() error {
	if c.Repo.Name == "" || c.Repo.URL == "" {
		return nil
	}
	var sout, serr bytes.Buffer
	args := []string{"repo", "add", c.Repo.Name, c.Repo.URL}
	cmd := exec.Command("helm", args...)
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("unable to add helm repo: %w: %s, %s", err, sout.String(), serr.String())
	}
	return nil
}

func (c *HelmChart) removeRepo() error {
	if c.Repo.Name == "" || c.Repo.URL == "" {
		return nil
	}
	var sout, serr bytes.Buffer
	args := []string{"repo", "remove", c.Repo.Name}
	cmd := exec.Command("helm", args...)
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("unable to remove repo: %w: %s, %s", err, sout.String(), serr.String())
	}
	return nil
}

// Logs fetches the logs from all pods managed by this release
// and prints them out.
func (c *HelmChart) Logs() error {
	kc := c.config.KubeClientSet
	podList, err := kc.CoreV1().Pods(c.Namespace).List(
		context.TODO(),
		metav1.ListOptions{LabelSelector: "app.kubernetes.io/instance=" + c.ReleaseName})
	if err != nil {
		return err
	}
	log.Logf("logs: found %d pods", len(podList.Items))
	tailLines := int64(200)
	for i := range podList.Items {
		pod := podList.Items[i]
		for _, con := range pod.Spec.Containers {
			for _, b := range []bool{true, false} {
				resp := kc.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
					Container: con.Name,
					Previous:  b,
					TailLines: &tailLines,
				}).Do(context.TODO())

				err := resp.Error()
				if err != nil {
					continue
				}

				logs, err := resp.Raw()
				if err != nil {
					continue
				}
				log.Logf("[%s]: %s", c.ReleaseName, string(logs))
			}
		}
	}
	return nil
}

func (c *HelmChart) HasVar(key, value string) bool {
	for _, v := range c.Vars {
		if v.Key == key && v.Value == value {
			return true
		}
	}
	return false
}
