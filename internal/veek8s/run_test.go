package veek8s

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const twoContext = `
apiVersion: v1
kind: Config
current-context: dev
clusters:
- name: cdev
  cluster:
    server: https://dev.example
    certificate-authority: /tmp/ca.crt
users:
- name: udev
  user:
    token: secret-token
contexts:
- name: dev
  context:
    cluster: cdev
    user: udev
    namespace: default
- name: dev-west
  context:
    cluster: cdev
    user: udev
`

func TestSwitch(t *testing.T) {
	a, out, errb, _ := newApp(t, twoContext)
	err := a.runArgs([]string{"nope"})
	if err == nil {
		t.Fatal("expected miss")
	}
	if out.Len() != 0 {
		t.Fatalf("miss wrote stdout %q", out.String())
	}
	if !strings.Contains(err.Error(), "dev-west") || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error %q", err.Error())
	}
	if _, err := os.Stat(filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s")); err == nil {
		t.Fatal("failed match created config")
	}

	out.Reset()
	errb.Reset()
	a.env["VEEK8S_CONTEXT"] = "previous"
	a.env["VEEK8S_SHELL"] = "abc123"
	if err := a.runArgs([]string{"dev"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "secret-token") {
		t.Fatal("token leaked to stdout")
	}
	applyExports(t, a, out.String())
	if a.env["VEEK8S_CONTEXT"] != "dev" || a.env["VEEK8S_PREV_CONTEXT"] != "previous" || a.env["VEEK8S_SHELL"] != "abc123" {
		t.Fatalf("env %+v", a.env)
	}
	body := readShell(t, a)
	if strings.Contains(body, "secret-token") == false || strings.Contains(body, "dev-west") {
		t.Fatalf("shell file:\n%s", body)
	}
	assertMode(t, shellPath(t, a), 0o600)

	out.Reset()
	if err := a.runArgs([]string{"-"}); err == nil || out.Len() != 0 {
		t.Fatalf("missing previous context err=%v out=%q", err, out.String())
	}
}

func TestSwitchPreviousAndPicker(t *testing.T) {
	a, out, _, _ := newApp(t, twoContext)
	a.env["VEEK8S_PREV_CONTEXT"] = "dev-west"
	a.env["VEEK8S_CONTEXT"] = "dev"
	if err := a.runArgs([]string{"-"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "VEEK8S_CONTEXT='dev-west'") {
		t.Fatalf("stdout %s", out.String())
	}
	if !strings.Contains(out.String(), "VEEK8S_PREV_CONTEXT='dev'") {
		t.Fatalf("prev not the context we left:\n%s", out.String())
	}

	a, out, _, _ = newApp(t, twoContext)
	err := a.runArgs([]string{"-"})
	if err == nil || out.Len() != 0 || !strings.Contains(err.Error(), "no previous") {
		t.Fatalf("empty prev err=%v out=%q", err, out.String())
	}

	a, out, _, _ = newApp(t, twoContext)
	a.env["VEEK8S_PREV_CONTEXT"] = "gone"
	if err := a.runArgs([]string{"-"}); err == nil || out.Len() != 0 {
		t.Fatal("missing prev")
	}

	a, out, _, _ = newApp(t, twoContext)
	a.pick = func(title string, items []string) (string, error) {
		if title != "context" || len(items) != 2 {
			t.Fatalf("pick %s %+v", title, items)
		}
		return "dev-west", nil
	}
	if err := a.runArgs([]string{"de"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "VEEK8S_CONTEXT='dev-west'") {
		t.Fatal(out.String())
	}

	a, out, _, cfg := newApp(t, twoContext)
	a.pick = func(string, []string) (string, error) { return "", errors.New("canceled") }
	if err := a.runArgs(nil); err == nil || out.Len() != 0 {
		t.Fatalf("cancel err=%v out=%q", err, out.String())
	}
	if _, err := os.Stat(filepath.Join(cfg, "veek8s", "shells")); err == nil {
		t.Fatal("cancel wrote a shell file")
	}

	a, out, _, _ = newApp(t, "")
	writeFile(t, filepath.Join(a.env["HOME"], ".kube", "config"), "apiVersion: v1\nkind: Config\n")
	a.env["KUBECONFIG"] = ""
	if err := a.runArgs(nil); err == nil || !strings.Contains(err.Error(), "no contexts") || out.Len() != 0 {
		t.Fatalf("empty %v out=%q", err, out.String())
	}
}

func TestNamespace(t *testing.T) {
	a, out, _, _ := newApp(t, twoContext)
	if err := a.runArgs([]string{"ns", "kube-system"}); err == nil || out.Len() != 0 {
		t.Fatalf("ns without vkx err=%v out=%q", err, out.String())
	}

	switchTo(t, a, "dev")
	a.look = func(name string) (string, error) {
		t.Fatalf("lookup %s", name)
		return "", errors.New("lookup")
	}
	out.Reset()
	if err := a.runArgs([]string{"ns", "kube-system"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "VEEK8S_PREV_NAMESPACE='default'") {
		t.Fatal(out.String())
	}
	if !strings.Contains(readShell(t, a), "namespace: kube-system") {
		t.Fatal(readShell(t, a))
	}
	applyExports(t, a, out.String())
	out.Reset()
	if err := a.runArgs([]string{"ns", "-"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "VEEK8S_PREV_NAMESPACE='kube-system'") {
		t.Fatal(out.String())
	}
	if !strings.Contains(readShell(t, a), "namespace: default") {
		t.Fatal(readShell(t, a))
	}

	before := readShell(t, a)
	a.look = func(string) (string, error) { return "", errors.New("missing") }
	out.Reset()
	if err := a.runArgs([]string{"ns"}); err == nil || out.Len() != 0 {
		t.Fatal("namespace menu without kubectl")
	}
	if readShell(t, a) != before {
		t.Fatal("failed menu changed the shell file")
	}

	a.look = func(string) (string, error) { return "kubectl", nil }
	a.output = func(name string, args []string) ([]byte, error) {
		if name != "kubectl" || !containsAll(args, "--kubeconfig", shellPath(t, a), "get", "ns") {
			t.Fatalf("kubectl %s %+v", name, args)
		}
		return []byte(`{"items":[{"metadata":{"name":"kube-system"}},{"metadata":{"name":"default"}}]}`), nil
	}
	a.pick = func(title string, items []string) (string, error) {
		if title != "namespace" || strings.Join(items, ",") != "kube-system,default" {
			t.Fatalf("pick %s %+v", title, items)
		}
		return "", errors.New("canceled")
	}
	if err := a.runArgs([]string{"ns"}); err == nil {
		t.Fatal("cancel")
	}
	if readShell(t, a) != before {
		t.Fatal("canceled menu wrote namespace")
	}
}

func TestNodesAndSSH(t *testing.T) {
	a, out, _, _ := newApp(t, twoContext)
	switchTo(t, a, "dev")
	nodes := []byte(nodeJSON)
	var kubectlCalls int
	a.look = func(name string) (string, error) {
		if name == "ssh" {
			return "", errors.New("missing ssh")
		}
		return name, nil
	}
	a.output = func(string, []string) ([]byte, error) {
		kubectlCalls++
		return nodes, nil
	}
	if err := a.runArgs([]string{"ssh"}); err == nil || !strings.Contains(err.Error(), "ssh not found") {
		t.Fatal(err)
	}
	if kubectlCalls != 0 {
		t.Fatal("ssh check happened after kubectl")
	}

	a.look = func(name string) (string, error) { return "/bin/" + name, nil }
	out.Reset()
	if err := a.runArgs([]string{"nodes"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "n1\t10.0.0.2\n") || strings.Contains(out.String(), "secret-token") {
		t.Fatalf("nodes %q", out.String())
	}

	var ran []string
	a.run = func(name string, args []string) error {
		ran = append([]string{name}, args...)
		return nil
	}
	var prompts []string
	a.prompt = func(p string) (string, error) {
		prompts = append(prompts, p)
		if strings.Contains(p, "user") {
			return "", nil
		}
		return "", nil
	}
	if err := a.runArgs([]string{"ssh", "n1"}); err == nil || len(ran) != 0 {
		t.Fatalf("empty user err=%v ran=%v", err, ran)
	}
	if _, err := os.Stat(filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s", "config.yaml")); err == nil {
		t.Fatal("empty user wrote config")
	}

	prompts = nil
	a.prompt = func(p string) (string, error) {
		prompts = append(prompts, p)
		if len(prompts) == 1 {
			return "root", nil
		}
		return "/keys/id", nil
	}
	if err := a.runArgs([]string{"ssh", "n1"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(ran, " ") != "/bin/ssh -p 22 -l root -i /keys/id 10.0.0.2" {
		t.Fatalf("ssh %v", ran)
	}
	cfgPath := filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s", "config.yaml")
	assertMode(t, cfgPath, 0o600)

	ran = nil
	prompts = nil
	if err := a.runArgs([]string{"ssh", "n2"}); err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 0 {
		t.Fatal("prompted again")
	}
	if strings.Join(ran, " ") != "/bin/ssh -p 22 -l root -i /keys/id n2.example" {
		t.Fatalf("second ssh %v", ran)
	}

	writeFile(t, cfgPath, "clusters:\n  dev:\n    username: root\n    address: nope\n")
	if err := a.runArgs([]string{"nodes"}); err == nil || !strings.Contains(err.Error(), "address") {
		t.Fatal(err)
	}
}

func TestSave(t *testing.T) {
	a, _, _, _ := newApp(t, twoContext)
	switchTo(t, a, "dev")
	writeFile(t, filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s", "config.yaml"), `
clusters:
  dev:
    username: root
    identityFile: /keys/id
    port: 2222
    address: internalIP
`)
	a.look = func(name string) (string, error) {
		if name == "veessh" {
			return "/bin/veessh", nil
		}
		if name == "kubectl" {
			return "/bin/kubectl", nil
		}
		return "", errors.New("missing")
	}
	a.output = func(string, []string) ([]byte, error) { return []byte(nodeJSON), nil }
	var showed, added []string
	a.quiet = func(name string, args []string) error {
		showed = append([]string{name}, args...)
		return exec.Command("false").Run()
	}
	a.run = func(name string, args []string) error {
		added = append([]string{name}, args...)
		return nil
	}
	if err := a.runArgs([]string{"save", "n1"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(showed, " ") != "/bin/veessh show dev--n1" {
		t.Fatalf("show %v", showed)
	}
	want := "/bin/veessh add dev--n1 --type ssh --host 10.0.0.2 --user root --port 2222 --group dev --tags kube --identity /keys/id"
	if strings.Join(added, " ") != want {
		t.Fatalf("add %v", added)
	}

	added = nil
	a.quiet = func(string, []string) error { return nil }
	if err := a.runArgs([]string{"save", "n1"}); err == nil || len(added) != 0 {
		t.Fatalf("overwrite guard err=%v", err)
	}
	if err := a.runArgs([]string{"save", "n1", "--overwrite"}); err != nil || len(added) == 0 {
		t.Fatalf("overwrite err=%v add=%v", err, added)
	}

	a.quiet = func(string, []string) error { return errors.New("veessh broken") }
	added = nil
	if err := a.runArgs([]string{"save", "n1"}); err == nil || len(added) != 0 {
		t.Fatal("broken veessh still added")
	}

	a.look = func(string) (string, error) { return "", errors.New("no") }
	if err := a.runArgs([]string{"save", "n1"}); err == nil || !strings.Contains(err.Error(), "veessh not found") {
		t.Fatal(err)
	}

	a.look = func(name string) (string, error) { return "/bin/" + name, nil }
	writeFile(t, filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s", "config.yaml"), "clusters: {}\n")
	if err := a.runArgs([]string{"save", "n1"}); err == nil || !strings.Contains(err.Error(), "run vssh") {
		t.Fatal(err)
	}
	if err := a.runArgs([]string{"save", "n"}); err == nil || !strings.Contains(err.Error(), "more than one") {
		t.Fatal(err)
	}
	if err := a.runArgs([]string{"save"}); err == nil {
		t.Fatal("save without node")
	}
}

func TestUsageAndGCCommand(t *testing.T) {
	a, out, errb, _ := newApp(t, twoContext)
	if err := a.runArgs([]string{"--help"}); err != nil || out.Len() != 0 {
		t.Fatalf("help err=%v out=%q", err, out.String())
	}
	if !strings.Contains(errb.String(), "vkx()") || !strings.Contains(errb.String(), "vns()") || !strings.Contains(errb.String(), "vssh()") {
		t.Fatal(errb.String())
	}
	for _, args := range [][]string{{"--nope"}, {"dev", "extra"}, {"ns", "a", "b"}, {"nodes", "x"}, {"save", "--wat", "n"}} {
		out.Reset()
		if err := a.runArgs(args); err == nil || out.Len() != 0 {
			t.Fatalf("%v err=%v out=%q", args, err, out.String())
		}
	}

	a.env["VEEK8S_SHELL"] = "../secret"
	out.Reset()
	if err := a.runArgs([]string{"dev"}); err == nil || out.Len() != 0 {
		t.Fatal("bad shell id")
	}

	dir := filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s", "shells")
	writeFile(t, filepath.Join(dir, "old.yaml"), "x")
	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "old.yaml"), past, past); err != nil {
		t.Fatal(err)
	}
	a.now = time.Now()
	a.env["VEEK8S_SHELL"] = ""
	errb.Reset()
	out.Reset()
	if err := a.runArgs([]string{"gc"}); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 || !strings.Contains(errb.String(), "removed 1") {
		t.Fatalf("gc out=%q err=%q", out.String(), errb.String())
	}
	root := filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s")
	if err := os.RemoveAll(filepath.Join(root, "shells")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "shells"), "not-a-directory")
	if err := a.runArgs([]string{"gc"}); err == nil {
		t.Fatal("gc should fail when shells is a file")
	}
}

func TestEdges(t *testing.T) {
	a, out, _, _ := newApp(t, twoContext)
	a.env["XDG_CONFIG_HOME"] = ""
	if err := a.runArgs([]string{"dev"}); err != nil {
		t.Fatal(err)
	}
	homeCfg := filepath.Join(a.env["HOME"], ".config", "veek8s", "shells")
	entries, err := os.ReadDir(homeCfg)
	if err != nil || len(entries) != 1 {
		t.Fatalf("default config dir %v %v", entries, err)
	}

	a, out, _, _ = newApp(t, twoContext)
	a.env["VEEK8S_SHELL"] = "abc"
	a.env["VEEK8S_CONTEXT"] = "dev"
	if err := a.runArgs([]string{"ns", "kube-system"}); err == nil || out.Len() != 0 {
		t.Fatalf("missing shell file err=%v out=%q", err, out.String())
	}
	a.env["VEEK8S_SHELL"] = "../x"
	a.look = func(string) (string, error) { return "ssh", nil }
	if err := a.runArgs([]string{"ssh", "n1"}); err == nil || !strings.Contains(err.Error(), "invalid VEEK8S_SHELL") {
		t.Fatal(err)
	}
	a.env["VEEK8S_SHELL"] = ""

	switchTo(t, a, "dev")
	writeFile(t, filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s", "config.yaml"), ":\n")
	a.look = func(string) (string, error) { return "kubectl", nil }
	if err := a.runArgs([]string{"nodes"}); err == nil {
		t.Fatal("corrupt cluster config")
	}

	writeFile(t, filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s", "config.yaml"), "clusters: {}\n")
	a.output = func(string, []string) ([]byte, error) { return []byte(`{"items":[]}`), nil }
	out.Reset()
	if err := a.runArgs([]string{"nodes"}); err == nil || !strings.Contains(err.Error(), "no nodes") || out.Len() != 0 {
		t.Fatalf("empty nodes err=%v out=%q", err, out.String())
	}
	if err := a.runArgs([]string{"ssh"}); err == nil {
		t.Fatal("ssh with no nodes")
	}
	a.output = func(string, []string) ([]byte, error) { return []byte("{"), nil }
	if err := a.runArgs([]string{"nodes"}); err == nil {
		t.Fatal("bad node json")
	}
	a.output = func(string, []string) ([]byte, error) { return []byte(nodeJSON), nil }
	a.look = func(name string) (string, error) {
		if name == "ssh" {
			return "/bin/ssh", nil
		}
		return "kubectl", nil
	}
	if err := a.runArgs([]string{"ssh", "n3"}); err == nil || !strings.Contains(err.Error(), "no address") {
		t.Fatal(err)
	}

	a.prompt = func(string) (string, error) { return "", errors.New("tty closed") }
	if err := a.runArgs([]string{"ssh", "n1"}); err == nil || !strings.Contains(err.Error(), "tty closed") {
		t.Fatal(err)
	}

	before := readShell(t, a)
	a.look = func(string) (string, error) { return "kubectl", nil }
	a.output = func(string, []string) ([]byte, error) { return []byte(`{"items":[]}`), nil }
	a.pick = func(string, []string) (string, error) {
		t.Fatal("empty namespace list should not pick")
		return "", nil
	}
	if err := a.runArgs([]string{"ns"}); err == nil || readShell(t, a) != before {
		t.Fatal(err)
	}

	a.home = func() (string, error) { return "", errors.New("no home") }
	a.env["XDG_CONFIG_HOME"] = ""
	a.env["KUBECONFIG"] = ""
	if err := a.runArgs([]string{"dev"}); err == nil || !strings.Contains(err.Error(), "no home") {
		t.Fatal(err)
	}
	a.env["XDG_CONFIG_HOME"] = filepath.Join(a.env["HOME"], "xdg")
	a.now = time.Time{}
	if err := a.runArgs([]string{"gc"}); err != nil {
		t.Fatal(err)
	}

	shell := shellPath(t, a)
	os.Remove(shell)
	if _, err := SetNamespace(shell, "x"); err == nil {
		t.Fatal("missing shell file")
	}
	multi := filepath.Join(t.TempDir(), "ab.yaml")
	writeFile(t, multi, "contexts:\n- name: a\n  context: {cluster: c}\n- name: b\n  context: {cluster: c}\n")
	if _, err := SetNamespace(multi, "x"); err == nil {
		t.Fatal("two contexts")
	}
}

func TestRunAndExec(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KUBECONFIG", filepath.Join(dir, "missing"))
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := Run([]string{"dev"}); err == nil {
		t.Fatal("missing kubeconfig")
	}
	oldErr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	if err := Run([]string{"--help"}); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stderr = oldErr
	help, _ := io.ReadAll(r)
	if !strings.Contains(string(help), "vssh()") {
		t.Fatalf("help %s", help)
	}

	script := filepath.Join(dir, "ssh")
	writeFile(t, script, "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runCmd(script, []string{"-p", "22"}); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "notdir")
	writeFile(t, file, "x")
	if _, err := GC(file, "", time.Now()); err == nil {
		t.Fatal("gc on a file")
	}
}

func TestChildStatus(t *testing.T) {
	err := exec.Command("false").Run()
	msg, code := Status(err)
	if msg != "" || code != 1 {
		t.Fatalf("child %q %d", msg, code)
	}
	msg, code = Status(errors.New("run vkx first"))
	if msg != "run vkx first" || code != 1 {
		t.Fatalf("plain %q %d", msg, code)
	}
	if msg, code = Status(nil); msg != "" || code != 0 {
		t.Fatalf("nil %q %d", msg, code)
	}
}

func TestExecHelpers(t *testing.T) {
	dir := t.TempDir()
	okPath := filepath.Join(dir, "ok")
	writeFile(t, okPath, "#!/bin/sh\nprintf '%s' \"$*\"\n")
	if err := os.Chmod(okPath, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := outputCmd(okPath, []string{"--kubeconfig", "file", "get", "nodes"})
	if err != nil || string(out) != "--kubeconfig file get nodes" {
		t.Fatalf("output %q %v", out, err)
	}
	bad := filepath.Join(dir, "bad")
	writeFile(t, bad, "#!/bin/sh\nexit 3\n")
	if err := os.Chmod(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = outputCmd(bad, nil)
	msg, code := Status(fmt.Errorf("kubectl: %w", err))
	if code != 3 || msg != "" {
		t.Fatalf("wrapped kubectl status msg=%q code=%d err=%v", msg, code, err)
	}
	if err := quietCmd(okPath, []string{"show", "p"}); err != nil {
		t.Fatal(err)
	}
	if err := quietCmd(bad, nil); err == nil {
		t.Fatal("quiet ignored exit")
	}
}

func TestReadLine(t *testing.T) {
	got, err := readLine(strings.NewReader("  root  \r\n"))
	if err != nil || got != "root" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = readLine(strings.NewReader("alice"))
	if err != nil || got != "alice" {
		t.Fatalf("eof %q %v", got, err)
	}
	if _, err := readLine(strings.NewReader("")); err == nil {
		t.Fatal("empty eof")
	}
}

func newApp(t *testing.T, kube string) (*app, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	home := t.TempDir()
	cfg := filepath.Join(home, "xdg")
	var stdout, stderr bytes.Buffer
	kubePath := ""
	if kube != "" {
		kubePath = filepath.Join(home, "kube.yaml")
		writeFile(t, kubePath, kube)
	}
	a := &app{
		env: map[string]string{
			"HOME":            home,
			"XDG_CONFIG_HOME": cfg,
			"KUBECONFIG":      kubePath,
		},
		stdout: &stdout,
		stderr: &stderr,
		now:    time.Now(),
		home:   func() (string, error) { return home, nil },
		pick: func(title string, items []string) (string, error) {
			t.Fatalf("unexpected pick %s %v", title, items)
			return "", nil
		},
		prompt: func(string) (string, error) {
			t.Fatal("unexpected prompt")
			return "", nil
		},
		look:   func(string) (string, error) { return "", errors.New("unexpected look") },
		output: func(string, []string) ([]byte, error) { t.Fatal("unexpected output"); return nil, nil },
		quiet:  func(string, []string) error { t.Fatal("unexpected quiet"); return nil },
		run:    func(string, []string) error { t.Fatal("unexpected run"); return nil },
	}
	return a, &stdout, &stderr, cfg
}

func switchTo(t *testing.T, a *app, name string) {
	t.Helper()
	var buf bytes.Buffer
	prev := a.stdout
	a.stdout = &buf
	if err := a.runArgs([]string{name}); err != nil {
		t.Fatal(err)
	}
	a.stdout = prev
	applyExports(t, a, buf.String())
}

func applyExports(t *testing.T, a *app, text string) {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimPrefix(line, "export ")
		if line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("export %q", line)
		}
		val = strings.Trim(val, "'")
		val = strings.ReplaceAll(val, `'\''`, `'`)
		a.env[key] = val
	}
}

func shellPath(t *testing.T, a *app) string {
	t.Helper()
	return filepath.Join(a.env["XDG_CONFIG_HOME"], "veek8s", "shells", a.env["VEEK8S_SHELL"]+".yaml")
}

func readShell(t *testing.T, a *app) string {
	t.Helper()
	b, err := os.ReadFile(shellPath(t, a))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func containsAll(args []string, want ...string) bool {
	got := strings.Join(args, " ")
	for _, w := range want {
		if !strings.Contains(got, w) {
			return false
		}
	}
	return true
}
