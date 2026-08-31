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

// tool-installer installs version-locked development tools.
package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const lockFile = "hack/tool-versions.json"

type lock struct {
	Assets      assets                `json:"assets"`
	Tools       map[string]tool       `json:"tools"`
	HelmPlugins map[string]helmPlugin `json:"helmPlugins"`
}

type assets struct {
	Envtest envtestAssets `json:"envtest"`
}

type envtestAssets struct {
	KubernetesVersion string `json:"kubernetesVersion"`
}

type installedVersion struct {
	SHA256  string `json:"sha256"`
	Version string `json:"version"`
}

type helmPlugin struct {
	Version    string `json:"version"`
	Repository string `json:"repository"`
}

type tool struct {
	Version    string              `json:"version"`
	Repository string              `json:"repository"`
	Format     string              `json:"format"`
	Platforms  map[string]platform `json:"platforms"`
}

type platform struct {
	Asset  string `json:"asset"`
	Binary string `json:"binary"`
	SHA256 string `json:"sha256"`
}

func main() {
	tools, err := readLock()
	if err != nil {
		fatalf("reading tool lock: %v", err)
	}
	if len(os.Args) == 2 {
		if os.Args[1] == "format-lock" {
			if err := formatJSON(lockFile); err != nil {
				fatalf("formatting %s: %v", lockFile, err)
			}
			return
		}
		if err := install(tools, os.Args[1]); err != nil {
			fatalf("installing %s: %v", os.Args[1], err)
		}
		return
	}
	if len(os.Args) == 3 {
		switch os.Args[1] {
		case "value":
			if os.Args[2] != "envtest-kubernetes-version" {
				fatalf("value %q is not present in %s", os.Args[2], lockFile)
			}
			if tools.Assets.Envtest.KubernetesVersion == "" {
				fatalf("value %q is empty in %s", os.Args[2], lockFile)
			}
			fmt.Println(tools.Assets.Envtest.KubernetesVersion)
			return
		case "helm-plugin":
			if err := installHelmPlugin(tools, os.Args[2]); err != nil {
				fatalf("installing Helm plugin %s: %v", os.Args[2], err)
			}
			return
		}
	}
	fatalf("usage: go run ./hack/tool-installer TOOL\n" +
		"       go run ./hack/tool-installer value envtest-kubernetes-version\n" +
		"       go run ./hack/tool-installer helm-plugin NAME\n" +
		"       go run ./hack/tool-installer format-lock")
}

func readLock() (lock, error) {
	data, err := os.ReadFile(lockFile)
	if err != nil {
		return lock{}, err
	}
	var tools lock
	if err := json.Unmarshal(data, &tools); err != nil {
		return lock{}, fmt.Errorf("parse %s: %w", lockFile, err)
	}
	return tools, nil
}

func formatJSON(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // trusted repository path
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("unexpected content after JSON document")
	}
	formatted, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	formatted = append(formatted, '\n')
	if bytes.Equal(data, formatted) {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	output, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	outputName := output.Name()
	defer func() { _ = os.Remove(outputName) }()
	if err := output.Chmod(info.Mode().Perm()); err != nil {
		_ = output.Close()
		return err
	}
	if _, err := output.Write(formatted); err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Rename(outputName, path)
}

func install(tools lock, name string) error {
	t, ok := tools.Tools[name]
	if !ok {
		return fmt.Errorf("tool %q is not present in %s", name, lockFile)
	}
	p, ok := t.Platforms[runtime.GOOS+"-"+runtime.GOARCH]
	if !ok {
		return fmt.Errorf("tool %q does not support %s/%s", name, runtime.GOOS, runtime.GOARCH)
	}
	if t.Version == "" || t.Repository == "" || p.Asset == "" || p.Binary == "" || p.SHA256 == "" {
		return errors.New("tool lock entry is incomplete")
	}

	installFolder := os.Getenv("INSTALL_FOLDER")
	if installFolder == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		installFolder = filepath.Join(workingDirectory, "bin")
	}
	destination := filepath.Join(installFolder, name)
	stateFile := filepath.Join(installFolder, ".installed-versions")
	wanted := installedVersion{Version: t.Version, SHA256: p.SHA256}
	state, err := readInstalledVersions(stateFile)
	if err != nil {
		return err
	}
	if state[name] == wanted {
		if info, err := os.Stat(destination); err == nil && !info.IsDir() { //nolint:gosec // trusted install folder and tool name
			fmt.Printf("%s %s already installed\n", name, t.Version)
			return nil
		}
	}

	// Migrate the former per-tool sidecar without downloading the tool again.
	legacyState := destination + ".version"
	if version, err := os.ReadFile(legacyState); err == nil && strings.TrimSpace(string(version)) == t.Version+" "+p.SHA256 { //nolint:gosec // trusted install folder and tool name
		if info, err := os.Stat(destination); err == nil && !info.IsDir() { //nolint:gosec // trusted install folder and tool name
			state[name] = wanted
			if err := writeInstalledVersions(stateFile, state); err != nil {
				return err
			}
			if err := os.Remove(legacyState); err != nil && !errors.Is(err, os.ErrNotExist) { //nolint:gosec // trusted install folder and tool name
				return err
			}
			fmt.Printf("%s %s already installed\n", name, t.Version)
			return nil
		}
	}

	asset := expand(p.Asset, t.Version)
	binary := expand(p.Binary, t.Version)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", t.Repository, t.Version, asset)

	tmp, err := os.CreateTemp("", name+"-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() //nolint:gosec // path was returned by os.CreateTemp
	defer func() { _ = tmp.Close() }()

	client := &http.Client{Timeout: 5 * time.Minute}
	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, response.Status)
	}

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hash), response.Body); err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	if actual := hex.EncodeToString(hash.Sum(nil)); actual != p.SHA256 {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", asset, actual, p.SHA256)
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil { //nolint:gosec // trusted Make target
		return err
	}
	output, err := os.CreateTemp(filepath.Dir(destination), "."+filepath.Base(destination)+"-*")
	if err != nil {
		return err
	}
	outputName := output.Name()
	defer func() { _ = os.Remove(outputName) }() //nolint:gosec // path was returned by os.CreateTemp

	switch t.Format {
	case "binary":
		_, err = io.Copy(output, tmp)
	case "tar.gz":
		err = extractFile(output, tmp, binary)
	default:
		err = fmt.Errorf("unsupported format %q", t.Format)
	}
	if err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Chmod(0o755); err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	if err := os.Rename(outputName, destination); err != nil { //nolint:gosec // trusted install folder and tool name
		return err
	}
	state[name] = wanted
	if err := writeInstalledVersions(stateFile, state); err != nil {
		return err
	}
	if err := os.Remove(legacyState); err != nil && !errors.Is(err, os.ErrNotExist) { //nolint:gosec // trusted install folder and tool name
		return err
	}
	fmt.Printf("%s %s installed successfully\n", name, t.Version)
	return nil
}

func readInstalledVersions(path string) (map[string]installedVersion, error) {
	data, err := os.ReadFile(path) //nolint:gosec // trusted install folder
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]installedVersion), nil
	}
	if err != nil {
		return nil, err
	}
	var state map[string]installedVersion
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if state == nil {
		state = make(map[string]installedVersion)
	}
	return state, nil
}

func writeInstalledVersions(path string, state map[string]installedVersion) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil { //nolint:gosec // trusted install folder
		return err
	}
	output, err := os.CreateTemp(filepath.Dir(path), ".installed-versions-*")
	if err != nil {
		return err
	}
	outputName := output.Name()
	defer func() { _ = os.Remove(outputName) }() //nolint:gosec // path was returned by os.CreateTemp
	if err := output.Chmod(0o600); err != nil {
		_ = output.Close()
		return err
	}
	if _, err := output.Write(data); err != nil {
		_ = output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Rename(outputName, path) //nolint:gosec // state path is derived from INSTALL_FOLDER
}

func installHelmPlugin(tools lock, name string) error {
	plugin, ok := tools.HelmPlugins[name]
	if !ok {
		return fmt.Errorf("Helm plugin %q is not present in %s", name, lockFile)
	}
	if plugin.Version == "" || plugin.Repository == "" {
		return errors.New("Helm plugin lock entry is incomplete")
	}

	output, err := run("helm", "plugin", "list")
	if err != nil {
		return err
	}
	installedVersion := helmPluginVersion(output, name)
	if installedVersion == plugin.Version {
		fmt.Printf("Helm plugin %s is already at %s\n", name, plugin.Version)
		return nil
	}
	if installedVersion != "" {
		if _, err := run("helm", "plugin", "remove", name); err != nil {
			return err
		}
	}
	_, err = run("helm", "plugin", "install", plugin.Repository, "--version", plugin.Version)
	return err
}

func helmPluginVersion(output, name string) string {
	for line := range strings.SplitSeq(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == name {
			return strings.TrimPrefix(fields[1], "v")
		}
	}
	return ""
}

func run(name string, args ...string) (string, error) {
	output, err := exec.Command(name, args...).CombinedOutput() //nolint:gosec // fixed executable and trusted lock data
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func extractFile(destination io.Writer, archive io.Reader, wanted string) error {
	compressed, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = compressed.Close() }()

	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%q not found in archive", wanted)
		}
		if err != nil {
			return err
		}
		if filepath.ToSlash(header.Name) == wanted && header.Typeflag == tar.TypeReg {
			const maxBinarySize = 512 << 20
			written, err := io.Copy(destination, io.LimitReader(reader, maxBinarySize+1))
			if err != nil {
				return err
			}
			if written > maxBinarySize {
				return fmt.Errorf("%q exceeds the maximum binary size", wanted)
			}
			return nil
		}
	}
}

func expand(value, version string) string {
	value = strings.ReplaceAll(value, "{{version}}", version)
	return strings.ReplaceAll(value, "{{version-no-v}}", strings.TrimPrefix(version, "v"))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
