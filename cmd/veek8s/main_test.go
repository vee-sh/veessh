package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIHelp(t *testing.T) {
	if os.Getenv("VEEK8S_CHILD") == "1" {
		os.Args = []string{"veek8s", "--help"}
		main()
		os.Exit(0)
	}
	out, errb, err := runChild(t, "TestCLIHelp", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 || !strings.Contains(errb.String(), "vkx()") {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}

func TestCLIMissPrintsStderr(t *testing.T) {
	if os.Getenv("VEEK8S_CHILD") == "1" {
		os.Args = []string{"veek8s", "missing"}
		main()
		os.Exit(0)
	}
	dir := t.TempDir()
	kube := filepath.Join(dir, "kube.yaml")
	if err := os.WriteFile(kube, []byte(`
apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: https://example.test
users:
- name: u
  user:
    token: secret-token
contexts:
- name: dev
  context:
    cluster: c
    user: u
`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errb, err := runChild(t, "TestCLIMissPrintsStderr", []string{
		"KUBECONFIG=" + kube,
		"XDG_CONFIG_HOME=" + filepath.Join(dir, "xdg"),
	})
	if err == nil {
		t.Fatal("expected exit 1")
	}
	ee, ok := err.(*exec.ExitError)
	if !ok || ee.ExitCode() != 1 {
		t.Fatalf("exit %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout %q", out.String())
	}
	if !strings.Contains(errb.String(), "dev") || strings.Contains(errb.String(), "secret-token") {
		t.Fatalf("stderr %q", errb.String())
	}
	if _, statErr := os.Stat(filepath.Join(dir, "xdg", "veek8s")); statErr == nil {
		t.Fatal("failed switch created a shell file")
	}
}

func runChild(t *testing.T, testName string, extra []string) (bytes.Buffer, bytes.Buffer, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^"+testName+"$")
	cmd.Env = append(os.Environ(), append([]string{"VEEK8S_CHILD=1"}, extra...)...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out, errb, err
}
