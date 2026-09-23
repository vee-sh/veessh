package veek8s

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// shellIDPattern blocks path characters in VEEK8S_SHELL.
var shellIDPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)

type rawConfig struct {
	CurrentContext string     `yaml:"current-context"`
	Clusters       []rawNamed `yaml:"clusters"`
	Users          []rawNamed `yaml:"users"`
	Contexts       []rawNamed `yaml:"contexts"`
}

type rawNamed struct {
	Name    string    `yaml:"name"`
	Cluster yaml.Node `yaml:"cluster"`
	User    yaml.Node `yaml:"user"`
	Context yaml.Node `yaml:"context"`
}

// Context is one kube context after a first-wins merge.
type Context struct {
	Cluster   string
	User      string
	Namespace string
	Body      map[string]any
}

// Kube is a merged set of kubeconfigs. The first file to define a name wins.
type Kube struct {
	Current  string
	Order    []string
	Contexts map[string]Context
	Clusters map[string]any
	Users    map[string]any
}

// KubeconfigPaths lists files to read. An empty value means ~/.kube/config.
func KubeconfigPaths(envValue, home string) ([]string, error) {
	if strings.TrimSpace(envValue) == "" {
		if strings.TrimSpace(home) == "" {
			return nil, errors.New("home directory is unknown")
		}
		return []string{filepath.Join(home, ".kube", "config")}, nil
	}
	var paths []string
	for _, p := range strings.Split(envValue, string(os.PathListSeparator)) {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return nil, errors.New("KUBECONFIG is empty")
	}
	return paths, nil
}

// Merge loads kubeconfig files. The first file to set a name or current-context wins.
func Merge(paths []string) (*Kube, error) {
	k := &Kube{
		Contexts: map[string]Context{},
		Clusters: map[string]any{},
		Users:    map[string]any{},
	}
	currentSet := false
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("kubeconfig %s: %w", path, err)
		}
		var raw rawConfig
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("kubeconfig %s: %w", path, err)
		}
		if !currentSet && strings.TrimSpace(raw.CurrentContext) != "" {
			k.Current = raw.CurrentContext
			currentSet = true
		}
		for _, c := range raw.Clusters {
			if c.Name == "" {
				continue
			}
			if _, ok := k.Clusters[c.Name]; ok {
				continue
			}
			v, err := decodeNode(c.Cluster)
			if err != nil {
				return nil, fmt.Errorf("cluster %s: %w", c.Name, err)
			}
			k.Clusters[c.Name] = v
		}
		for _, u := range raw.Users {
			if u.Name == "" {
				continue
			}
			if _, ok := k.Users[u.Name]; ok {
				continue
			}
			v, err := decodeNode(u.User)
			if err != nil {
				return nil, fmt.Errorf("user %s: %w", u.Name, err)
			}
			k.Users[u.Name] = v
		}
		for _, c := range raw.Contexts {
			if c.Name == "" {
				continue
			}
			if _, ok := k.Contexts[c.Name]; ok {
				continue
			}
			bodyAny, err := decodeNode(c.Context)
			if err != nil {
				return nil, fmt.Errorf("context %s: %w", c.Name, err)
			}
			body, ok := bodyAny.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("context %s: expected a mapping", c.Name)
			}
			cluster, _ := body["cluster"].(string)
			user, _ := body["user"].(string)
			ns, _ := body["namespace"].(string)
			k.Contexts[c.Name] = Context{Cluster: cluster, User: user, Namespace: ns, Body: body}
			k.Order = append(k.Order, c.Name)
		}
	}
	return k, nil
}

func decodeNode(n yaml.Node) (any, error) {
	if n.Kind == 0 {
		return nil, errors.New("missing value")
	}
	var v any
	if err := n.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// Shell returns a kubeconfig containing only name, its cluster, and its user.
func (k *Kube) Shell(name string) ([]byte, error) {
	c, ok := k.Contexts[name]
	if !ok {
		return nil, fmt.Errorf("context %q not found", name)
	}
	if c.Cluster == "" {
		return nil, fmt.Errorf("context %q has no cluster", name)
	}
	cluster, ok := k.Clusters[c.Cluster]
	if !ok {
		return nil, fmt.Errorf("context %q: cluster %q not found", name, c.Cluster)
	}
	doc := map[string]any{
		"apiVersion":      "v1",
		"kind":            "Config",
		"current-context": name,
		"contexts": []any{
			map[string]any{"name": name, "context": c.Body},
		},
		"clusters": []any{
			map[string]any{"name": c.Cluster, "cluster": cluster},
		},
	}
	if c.User != "" {
		user, ok := k.Users[c.User]
		if !ok {
			return nil, fmt.Errorf("context %q: user %q not found", name, c.User)
		}
		doc["users"] = []any{map[string]any{"name": c.User, "user": user}}
	}
	return yaml.Marshal(doc)
}

// Match returns names containing query. An exact name wins over other substrings.
// ponytail: exact wins so `vkx prod` does not prompt when prod and prod-west both exist.
func Match(names []string, query string) []string {
	if query == "" {
		out := make([]string, len(names))
		copy(out, names)
		return out
	}
	var exact, partial []string
	for _, n := range names {
		if n == query {
			exact = append(exact, n)
			continue
		}
		if strings.Contains(n, query) {
			partial = append(partial, n)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return partial
}

// ShellID reuses an existing id or generates one. Ids are alphanumeric so they cannot be paths.
func ShellID(existing string) (string, error) {
	if existing == "" {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		return hex.EncodeToString(b[:]), nil
	}
	if !shellIDPattern.MatchString(existing) {
		return "", fmt.Errorf("invalid VEEK8S_SHELL %q", existing)
	}
	return existing, nil
}

// WriteAtomic writes dir/id.yaml mode 0600 via a temp file in the same directory.
func WriteAtomic(dir, id string, data []byte) (string, error) {
	if !shellIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid shell id %q", id)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	final := filepath.Join(dir, id+".yaml")
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	done := false
	defer func() {
		if !done {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, final); err != nil {
		return "", err
	}
	done = true
	if err := os.Chmod(final, 0o600); err != nil {
		return "", err
	}
	return final, nil
}

// SetNamespace sets namespace in a shell kubeconfig and returns the previous value.
// An empty ns clears it. A parse error leaves the file unchanged.
func SetNamespace(path, ns string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	contexts, _ := doc["contexts"].([]any)
	if len(contexts) != 1 {
		return "", errors.New("shell kubeconfig has no context")
	}
	entry, _ := contexts[0].(map[string]any)
	if entry == nil {
		return "", errors.New("shell kubeconfig has no context")
	}
	body, _ := entry["context"].(map[string]any)
	if body == nil {
		return "", errors.New("shell kubeconfig has no context")
	}
	prev, _ := body["namespace"].(string)
	if ns == "" {
		delete(body, "namespace")
	} else {
		body["namespace"] = ns
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	id := strings.TrimSuffix(filepath.Base(path), ".yaml")
	if _, err := WriteAtomic(filepath.Dir(path), id, out); err != nil {
		return "", err
	}
	return prev, nil
}

// Exports is the only stdout vkx produces.
func Exports(kubeconfig, id, context, prevContext string) string {
	return strings.Join([]string{
		"export KUBECONFIG=" + shellQuote(kubeconfig),
		"export VEEK8S_SHELL=" + shellQuote(id),
		"export VEEK8S_CONTEXT=" + shellQuote(context),
		"export VEEK8S_PREV_CONTEXT=" + shellQuote(prevContext),
		"",
	}, "\n")
}

// NamespaceExports adds the previous namespace. The other lines match Exports.
func NamespaceExports(kubeconfig, id, context, prevContext, prevNamespace string) string {
	return Exports(kubeconfig, id, context, prevContext) +
		"export VEEK8S_PREV_NAMESPACE=" + shellQuote(prevNamespace) + "\n"
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ProfileName is the veessh profile for a node. Slashes and colons become dashes.
func ProfileName(context, node string) string {
	r := strings.NewReplacer("/", "-", ":", "-")
	return r.Replace(context) + "--" + r.Replace(node)
}

// GC deletes files in dir older than 24h. keepID's yaml is kept even if it is old.
func GC(dir, keepID string, now time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	keep := ""
	if keepID != "" && shellIDPattern.MatchString(keepID) {
		keep = keepID + ".yaml"
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == keep {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return n, err
		}
		if now.Sub(info.ModTime()) <= 24*time.Hour {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
