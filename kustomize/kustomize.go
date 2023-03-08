package kustomize

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/cresta/magehelper/files"
	"gopkg.in/yaml.v3"
)

var YAMLSeparator = regexp.MustCompile("---\n")
var namespacelessKinds = map[string]bool{
	"ClusterRole":        true,
	"ClusterRoleBinding": true,
	"StorageClass":       true,
	"PersistentVolume":   true,
}

// Manifest represents a Kubernetes manifest YAML file
type Manifest struct {
	// YAML is the original YAML string
	YAML string
	// Parsed is the YAML string parsed as a map
	Parsed map[string]any
	// Name is the name of the manifest
	Name string
	// Namespace is the Kubernetes namespace this manifest is defined in.
	Namespace string
	// Kind is the kind of the manifest
	Kind string
}

// ShortName returns a short display name of the Manifest, including on ly the kind and name
func (m *Manifest) ShortName() string {
	return fmt.Sprintf("[%s] %s", m.Kind, m.Name)
}

// LongName returns a long display name for the Manifest, including namespace, kind and name
func (m *Manifest) LongName() string {
	return m.Namespace + "-" + m.Kind + "-" + m.Name
}

// Init runs the `kuztomize init` command on a specified directory, returning the kustomize generated manifest.
//
// The returned map maps the Manifest long name to the manifest.
func Init(ctx context.Context, path string) (map[string]Manifest, error) {
	if !files.FileExists(filepath.Join(path, "kustomization.yaml")) {
		_, err := shell(path, "kustomize", "init", ".", "--recursive", "--autodetect")
		if err != nil {
			return nil, fmt.Errorf("cannot init kustomize %s: %w", path, err)
		}
	}
	output, err := shell(path, "kustomize", "build", "--load-restrictor=LoadRestrictionsNone")
	if err != nil {
		return nil, fmt.Errorf("failed to run kustomize at %s: %w", path, err)
	}
	yamls := YAMLSeparator.Split(string(output), -1)
	documents := make(map[string]Manifest, len(yamls))
	for _, content := range yamls {
		var parsed map[string]any
		err := yaml.Unmarshal([]byte(content), &parsed)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal output:\n%s\n%w", content, err)
		}
		kind, ok := getString(parsed, true, "kind")
		if !ok {
			return nil, fmt.Errorf("missing kind in yaml document:\n%s\n", content)
		}
		name, ok := getString(parsed, true, "metadata", "name")
		if !ok {
			return nil, fmt.Errorf("missing metadata.name in yaml document:\n%s\n", content)
		}
		namespace := ""
		if kind == "Namespace" {
			namespace = name
		} else {
			namespace, ok = getString(parsed, true, "metadata", "namespace")
			if !ok && !namespacelessKinds[kind] {
				return nil, fmt.Errorf("missing metadata.namespace in yaml document:\n%s\n", content)
			}
		}
		if namespace == "" {
			namespace = "[no-namespace]"
		}
		m := Manifest{
			YAML:      content,
			Parsed:    parsed,
			Name:      name,
			Namespace: namespace,
			Kind:      kind,
		}
		documents[m.LongName()] = m
	}
	return documents, nil
}

func shell(pwd string, cmdName string, args ...string) (stdout string, err error) {
	cmd := exec.Command(cmdName, args...)
	cmd.Dir = pwd
	stderr := bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%w. Stderr: %s", err, stderr.String())
	}
	return string(output), nil
}

func getString(m map[string]any, verbose bool, keys ...string) (string, bool) {
	valueAny, ok := getValue(m, verbose, keys...)
	if !ok {
		return "", ok
	}
	value, ok := valueAny.(string)
	return value, ok
}

func getValue(m map[string]any, verbose bool, keys ...string) (any, bool) {
	var v any = m
	for _, key := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			if verbose {
				fmt.Printf("not a map: %v\n", v)
			}
			return "", false
		}
		v, ok = m[key]
		if !ok {
			if verbose {
				fmt.Printf("%s not in %v\n", key, m)
			}
			return "", false
		}
	}
	return v, true
}
