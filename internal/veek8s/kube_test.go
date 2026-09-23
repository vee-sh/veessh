package veek8s

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMergeMinify(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yaml")
	b := filepath.Join(dir, "b.yaml")
	writeFile(t, a, `
apiVersion: v1
kind: Config
current-context: a
clusters:
- name: ca
  cluster:
    server: https://a.example
    certificate-authority: /tmp/a.crt
    insecure-skip-tls-verify: false
- name: shared-c
  cluster:
    server: https://shared-a
users:
- name: ua
  user:
    token: a-token
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args: ["eks", "get-token"]
contexts:
- name: a
  context:
    cluster: ca
    user: ua
    namespace: ns-a
- name: shared
  context:
    cluster: shared-c
    user: ua
- name: bare
  context:
    cluster: ca
`)
	writeFile(t, b, `
apiVersion: v1
kind: Config
current-context: b
clusters:
- name: shared-c
  cluster:
    server: https://shared-b
- name: cb
  cluster:
    server: https://b.example
users:
- name: ub
  user:
    token: b-token
- name: ua
  user:
    token: second-user-loses
contexts:
- name: shared
  context:
    cluster: shared-c
    user: ub
    namespace: from-b
- name: b
  context:
    cluster: cb
    user: ub
`)

	k, err := Merge([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if k.Current != "a" {
		t.Fatalf("current %q", k.Current)
	}
	if got := strings.Join(k.Order, ","); got != "a,shared,bare,b" {
		t.Fatalf("order %s", got)
	}
	if k.Contexts["shared"].User != "ua" || k.Contexts["shared"].Namespace != "" {
		t.Fatalf("shared %#v", k.Contexts["shared"])
	}

	shells := filepath.Join(dir, "shells")
	if _, err := k.Shell("nope"); err == nil {
		t.Fatal("missing context succeeded")
	}
	if _, err := os.Stat(shells); !os.IsNotExist(err) {
		t.Fatal("failed shell wrote a directory")
	}

	data, err := k.Shell("shared")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "a-token") == false {
		t.Fatal("expected first user token")
	}
	if strings.Contains(string(data), "b-token") || strings.Contains(string(data), "https://shared-b") || strings.Contains(string(data), "second-user-loses") {
		t.Fatalf("second file leaked:\n%s", data)
	}
	assertShell(t, data, "shared", "https://shared-a", "aws")

	bare, err := k.Shell("bare")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bare), "users:") {
		t.Fatalf("empty user should omit users:\n%s", bare)
	}

	path, err := WriteAtomic(shells, "abc", data)
	if err != nil {
		t.Fatal(err)
	}
	assertMode(t, path, 0o600)
	assertMode(t, shells, 0o700)

	replaced, err := k.Shell("a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteAtomic(shells, "abc", replaced); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "https://a.example") == false || strings.Contains(string(got), "https://shared-a") {
		t.Fatalf("replace failed:\n%s", got)
	}
	if strings.Contains(string(got), "insecure-skip-tls-verify") == false {
		t.Fatalf("tls flag dropped:\n%s", got)
	}

	if _, err := WriteAtomic(shells, "../abc", []byte("x")); err == nil {
		t.Fatal("path id accepted")
	}
	if _, err := WriteAtomic(shells, "", []byte("x")); err == nil {
		t.Fatal("empty id accepted")
	}

	prev, err := SetNamespace(path, "kube-system")
	if err != nil {
		t.Fatal(err)
	}
	if prev != "ns-a" {
		t.Fatalf("prev namespace %q", prev)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "namespace: kube-system") || !strings.Contains(string(updated), "https://a.example") {
		t.Fatalf("namespace rewrite lost data:\n%s", updated)
	}
	assertMode(t, path, 0o600)
	cleared, err := SetNamespace(path, "")
	if err != nil || cleared != "kube-system" {
		t.Fatalf("clear prev=%q err=%v", cleared, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "namespace:") {
		t.Fatalf("namespace not cleared:\n%s", after)
	}

	bad := filepath.Join(shells, "bad.yaml")
	writeFile(t, bad, ":\n: not yaml: [")
	before, _ := os.ReadFile(bad)
	if _, err := SetNamespace(bad, "x"); err == nil {
		t.Fatal("bad yaml accepted")
	}
	afterBad, _ := os.ReadFile(bad)
	if string(before) != string(afterBad) {
		t.Fatal("bad yaml was rewritten")
	}
}

func TestMergeErrors(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	if _, err := Merge([]string{missing}); err == nil {
		t.Fatal("missing file")
	}
	bad := filepath.Join(dir, "bad.yaml")
	writeFile(t, bad, ":\n: not yaml: [")
	if _, err := Merge([]string{bad}); err == nil {
		t.Fatal("bad yaml")
	}
	empty := filepath.Join(dir, "empty.yaml")
	writeFile(t, empty, "")
	k, err := Merge([]string{empty})
	if err != nil || len(k.Order) != 0 {
		t.Fatalf("empty file: %+v %v", k, err)
	}
	notMap := filepath.Join(dir, "nomap.yaml")
	writeFile(t, notMap, `
contexts:
- name: x
  context: just-a-string
`)
	if _, err := Merge([]string{notMap}); err == nil {
		t.Fatal("context string accepted")
	}
	noCluster := filepath.Join(dir, "nocluster.yaml")
	writeFile(t, noCluster, `
clusters:
- name: c
contexts:
- name: x
  context:
    cluster: c
    user: u
`)
	if _, err := Merge([]string{noCluster}); err == nil {
		t.Fatal("cluster without a value accepted")
	}
	dangling := filepath.Join(dir, "dangling.yaml")
	writeFile(t, dangling, `
clusters:
- name: c
  cluster:
    server: https://c
contexts:
- name: x
  context:
    cluster: missing
    user: u
users:
- name: u
  user:
    token: t
`)
	k, err = Merge([]string{dangling})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.Shell("x"); err == nil {
		t.Fatal("dangling cluster accepted")
	}
	noUser := filepath.Join(dir, "nouser.yaml")
	writeFile(t, noUser, `
clusters:
- name: c
  cluster:
    server: https://c
contexts:
- name: x
  context:
    cluster: c
    user: gone
`)
	k, err = Merge([]string{noUser})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.Shell("x"); err == nil {
		t.Fatal("missing user accepted")
	}
}

func TestMatchAndNames(t *testing.T) {
	if got := Match([]string{"prod", "prod-west"}, "prod"); len(got) != 1 || got[0] != "prod" {
		t.Fatalf("exact %+v", got)
	}
	if got := Match([]string{"prod", "prod-west"}, "pro"); len(got) != 2 {
		t.Fatalf("substring %+v", got)
	}
	if got := Match([]string{"Dev"}, "dev"); len(got) != 0 {
		t.Fatalf("case %+v", got)
	}
	all := []string{"a", "b"}
	got := Match(all, "")
	got[0] = "z"
	if all[0] != "a" {
		t.Fatal("Match aliased the input slice")
	}
	if ProfileName("arn:aws:eks/c", "node/1") != "arn-aws-eks-c--node-1" {
		t.Fatal(ProfileName("arn:aws:eks/c", "node/1"))
	}
	id, err := ShellID("")
	if err != nil || !shellIDPattern.MatchString(id) {
		t.Fatalf("generated id %q %v", id, err)
	}
	if _, err := ShellID("../etc"); err == nil {
		t.Fatal("path shell id")
	}
	if got, err := ShellID("abc"); err != nil || got != "abc" {
		t.Fatalf("reuse %q %v", got, err)
	}
}

func TestExportsQuote(t *testing.T) {
	got := Exports("/tmp/it's.yaml", "abc", "dev", "old")
	want := []string{
		`export KUBECONFIG='/tmp/it'\''s.yaml'`,
		`export VEEK8S_SHELL='abc'`,
		`export VEEK8S_CONTEXT='dev'`,
		`export VEEK8S_PREV_CONTEXT='old'`,
		"",
	}
	if got != strings.Join(want, "\n") {
		t.Fatalf("exports:\n%s", got)
	}
	base := Exports("/k", "abc", "dev", "old")
	ns := NamespaceExports("/k", "abc", "dev", "old", "kube-system")
	if !strings.HasPrefix(ns, base) || !strings.HasSuffix(ns, "export VEEK8S_PREV_NAMESPACE='kube-system'\n") {
		t.Fatalf("ns exports:\n%s", ns)
	}
}

func TestKubeconfigPaths(t *testing.T) {
	got, err := KubeconfigPaths("", "/home/a")
	if err != nil || len(got) != 1 || got[0] != filepath.Join("/home/a", ".kube", "config") {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := KubeconfigPaths("   ", ""); err == nil {
		t.Fatal("blank env and home")
	}
	sep := string(os.PathListSeparator)
	if _, err := KubeconfigPaths(sep+sep, "/home/a"); err == nil {
		t.Fatal("empty list")
	}
	got, err = KubeconfigPaths("  /a  "+sep+" /b ", "/home/a")
	if err != nil || len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("%v %v", got, err)
	}
}

func TestGC(t *testing.T) {
	if n, err := GC(filepath.Join(t.TempDir(), "missing"), "abc", time.Now()); err != nil || n != 0 {
		t.Fatalf("missing dir n=%d err=%v", n, err)
	}
	dir := t.TempDir()
	old := filepath.Join(dir, "old.yaml")
	fresh := filepath.Join(dir, "fresh.yaml")
	active := filepath.Join(dir, "abc.yaml")
	sub := filepath.Join(dir, "subdir")
	writeFile(t, old, "old")
	writeFile(t, fresh, "fresh")
	writeFile(t, active, "active")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	past := now.Add(-48 * time.Hour)
	boundary := now.Add(-24 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(active, past, past); err != nil {
		t.Fatal(err)
	}
	bound := filepath.Join(dir, "bound.yaml")
	writeFile(t, bound, "bound")
	if err := os.Chtimes(bound, boundary, boundary); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(dir), "veek8s-secret-"+t.Name()+".yaml")
	writeFile(t, outside, "secret")
	t.Cleanup(func() { os.Remove(outside) })

	n, err := GC(dir, "abc", now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("removed %d", n)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("old file kept")
	}
	if _, err := os.Stat(active); err != nil {
		t.Fatal("active shell file was removed")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("fresh removed")
	}
	if _, err := os.Stat(bound); err != nil {
		t.Fatal("24h boundary removed")
	}
	if _, err := os.Stat(sub); err != nil {
		t.Fatal("directory removed")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("gc escaped the directory")
	}

	n, err = GC(dir, "../secret", now)
	if err != nil || n != 1 {
		t.Fatalf("bad keep id n=%d err=%v", n, err)
	}
	if _, err := os.Stat(active); !os.IsNotExist(err) {
		t.Fatal("invalid keep id protected the old shell file")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("bad keep id removed a file outside the directory")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatal("fresh removed with invalid keep id")
	}
	if _, err := os.Stat(bound); err != nil {
		t.Fatal("boundary removed with invalid keep id")
	}
}

func TestFuzzDoesNotPanic(t *testing.T) {
	// Seed coverage lives in the fuzz tests. This runs one hostile input in normal tests.
	dir := t.TempDir()
	p := filepath.Join(dir, "k")
	writeFile(t, p, ":\x00\xff not yaml")
	_, _ = Merge([]string{p})
	_ = shellQuote("a'\x00b")
	_ = Match([]string{"a", "b"}, "\x00")
}

func FuzzMerge(f *testing.F) {
	f.Add([]byte("apiVersion: v1\nkind: Config\n"))
	f.Add([]byte(":\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		p := filepath.Join(dir, "k")
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
		k, err := Merge([]string{p})
		if err != nil || k == nil {
			return
		}
		for _, name := range k.Order {
			_, _ = k.Shell(name)
		}
	})
}

func FuzzShellQuote(f *testing.F) {
	f.Add("plain")
	f.Add("a'b")
	f.Fuzz(func(t *testing.T, s string) {
		q := shellQuote(s)
		if len(q) < 2 || q[0] != '\'' || q[len(q)-1] != '\'' {
			t.Fatalf("unquoted %q", q)
		}
	})
}

func assertMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != mode {
		t.Fatalf("%s mode %o", path, st.Mode().Perm())
	}
}

func assertShell(t *testing.T, data []byte, context, server, execCmd string) {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["apiVersion"] != "v1" || doc["kind"] != "Config" || doc["current-context"] != context {
		t.Fatalf("header %+v", doc)
	}
	contexts := doc["contexts"].([]any)
	if len(contexts) != 1 || contexts[0].(map[string]any)["name"] != context {
		t.Fatalf("contexts %+v", contexts)
	}
	clusters := doc["clusters"].([]any)
	if len(clusters) != 1 {
		t.Fatalf("clusters %+v", clusters)
	}
	cl := clusters[0].(map[string]any)["cluster"].(map[string]any)
	if cl["server"] != server {
		t.Fatalf("server %+v", cl["server"])
	}
	if execCmd == "" {
		return
	}
	users := doc["users"].([]any)
	user := users[0].(map[string]any)["user"].(map[string]any)
	ex := user["exec"].(map[string]any)
	if ex["command"] != execCmd {
		t.Fatalf("exec %+v", ex)
	}
}
