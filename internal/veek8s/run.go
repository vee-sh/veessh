package veek8s

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const helpText = `veek8s — per-shell kube context

  veek8s [context]          switch context, print exports
  veek8s ns [namespace]     set namespace in this shell
  veek8s ssh [node]         ssh to a node
  veek8s nodes              list nodes
  veek8s save <node>        save a node as a veessh profile (--overwrite)
  veek8s gc                 delete shell files older than 24h

  vkx() { local out; out=$(veek8s "$@") || return; eval "$out"; }
  vns() { local out; out=$(veek8s ns "$@") || return; eval "$out"; }
  vssh() { veek8s ssh "$@"; }
`

// Status splits an error into a stderr line and a process exit code.
// Child processes keep their own message and only the exit code is returned.
func Status(err error) (message string, code int) {
	if err == nil {
		return "", 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return "", ee.ExitCode()
	}
	return err.Error(), 1
}

type app struct {
	env    map[string]string
	getenv func(string) string
	stdout io.Writer
	stderr io.Writer
	now    time.Time
	home   func() (string, error)
	pick   func(title string, items []string) (string, error)
	prompt func(string) (string, error)
	look   func(string) (string, error)
	output func(name string, args []string) ([]byte, error)
	quiet  func(name string, args []string) error
	run    func(name string, args []string) error
}

// Run is the veek8s entrypoint. Exports go to stdout. Errors return.
func Run(args []string) error {
	return (&app{
		getenv: os.Getenv,
		stdout: os.Stdout,
		stderr: os.Stderr,
		now:    time.Now(),
		home:   os.UserHomeDir,
		pick:   Pick,
		prompt: promptLine,
		look:   exec.LookPath,
		output: outputCmd,
		quiet:  quietCmd,
		run:    runCmd,
	}).runArgs(args)
}

func (a *app) get(key string) string {
	if a.env != nil {
		return a.env[key]
	}
	if a.getenv != nil {
		return a.getenv(key)
	}
	return os.Getenv(key)
}

func (a *app) runArgs(args []string) error {
	if len(args) == 0 {
		return a.cmdContext("")
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(a.stderr, helpText)
		return nil
	case "ns":
		if len(args) > 2 {
			return errors.New("usage: veek8s ns [namespace]")
		}
		q := ""
		if len(args) == 2 {
			q = args[1]
		}
		return a.cmdNamespace(q)
	case "ssh":
		if len(args) > 2 {
			return errors.New("usage: veek8s ssh [node]")
		}
		q := ""
		if len(args) == 2 {
			q = args[1]
		}
		return a.cmdSSH(q)
	case "nodes":
		if len(args) > 1 {
			return errors.New("usage: veek8s nodes")
		}
		return a.cmdNodes()
	case "save":
		return a.cmdSave(args[1:])
	case "gc":
		if len(args) > 1 {
			return errors.New("usage: veek8s gc")
		}
		return a.cmdGC()
	default:
		if strings.HasPrefix(args[0], "-") && args[0] != "-" {
			return fmt.Errorf("unknown flag %s", args[0])
		}
		if len(args) > 1 {
			return errors.New("usage: veek8s [context]")
		}
		return a.cmdContext(args[0])
	}
}

func (a *app) cmdContext(query string) error {
	k, err := a.loadKube()
	if err != nil {
		return err
	}
	if len(k.Order) == 0 {
		return errors.New("no contexts")
	}
	name, err := a.choose(k.Order, query, "context", a.get("VEEK8S_PREV_CONTEXT"))
	if err != nil {
		return err
	}
	data, err := k.Shell(name)
	if err != nil {
		return err
	}
	id, err := ShellID(a.get("VEEK8S_SHELL"))
	if err != nil {
		return err
	}
	root, err := a.configRoot()
	if err != nil {
		return err
	}
	path, err := WriteAtomic(filepath.Join(root, "shells"), id, data)
	if err != nil {
		return err
	}
	fmt.Fprint(a.stdout, Exports(path, id, name, a.get("VEEK8S_CONTEXT")))
	return nil
}

func (a *app) cmdNamespace(query string) error {
	path, ctx, err := a.shellFile()
	if err != nil {
		return err
	}
	var ns string
	switch {
	case query == "-":
		ns = a.get("VEEK8S_PREV_NAMESPACE")
	case query == "":
		names, err := a.kubectlNames(path, "ns")
		if err != nil {
			return err
		}
		ns, err = a.pick("namespace", names)
		if err != nil {
			return err
		}
	default:
		ns = query
	}
	prev, err := SetNamespace(path, ns)
	if err != nil {
		return err
	}
	fmt.Fprint(a.stdout, NamespaceExports(path, a.get("VEEK8S_SHELL"), ctx, a.get("VEEK8S_PREV_CONTEXT"), prev))
	return nil
}

func (a *app) cmdNodes() error {
	path, ctx, err := a.shellFile()
	if err != nil {
		return err
	}
	nodes, err := a.loadNodes(path, ctx)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return errors.New("no nodes")
	}
	for _, n := range nodes {
		fmt.Fprintf(a.stdout, "%s\t%s\n", n.Name, n.Addr)
	}
	return nil
}

func (a *app) cmdSSH(query string) error {
	if _, err := a.look("ssh"); err != nil {
		return errors.New("ssh not found on PATH")
	}
	path, ctx, err := a.shellFile()
	if err != nil {
		return err
	}
	nodes, err := a.loadNodes(path, ctx)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return errors.New("no nodes")
	}
	names := make([]string, len(nodes))
	byName := map[string]node{}
	for i, n := range nodes {
		names[i] = n.Name
		if _, ok := byName[n.Name]; !ok {
			byName[n.Name] = n
		}
	}
	name, err := a.choose(names, query, "node", "")
	if err != nil {
		return err
	}
	n := byName[name]
	if n.Addr == "" {
		return fmt.Errorf("node %s has no address", name)
	}
	cluster, err := a.ensureCluster(ctx)
	if err != nil {
		return err
	}
	port := cluster.Port
	if port <= 0 {
		port = 22
	}
	sshPath, err := a.look("ssh")
	if err != nil {
		return errors.New("ssh not found on PATH")
	}
	args := []string{"-p", strconv.Itoa(port), "-l", cluster.Username}
	if cluster.IdentityFile != "" {
		args = append(args, "-i", cluster.IdentityFile)
	}
	args = append(args, n.Addr)
	return a.run(sshPath, args)
}

func (a *app) cmdSave(args []string) error {
	overwrite := false
	query := ""
	for _, arg := range args {
		if arg == "--overwrite" {
			overwrite = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("unknown flag %s", arg)
		}
		if query != "" {
			return errors.New("usage: veek8s save <node> [--overwrite]")
		}
		query = arg
	}
	if query == "" {
		return errors.New("usage: veek8s save <node> [--overwrite]")
	}
	bin, err := a.look("veessh")
	if err != nil {
		return errors.New("veessh not found on PATH")
	}
	path, ctx, err := a.shellFile()
	if err != nil {
		return err
	}
	nodes, err := a.loadNodes(path, ctx)
	if err != nil {
		return err
	}
	names := make([]string, len(nodes))
	byName := map[string]node{}
	for i, n := range nodes {
		names[i] = n.Name
		if _, ok := byName[n.Name]; !ok {
			byName[n.Name] = n
		}
	}
	matches := Match(names, query)
	switch len(matches) {
	case 0:
		return fmt.Errorf("no node matching %q\n%s", query, strings.Join(names, "\n"))
	case 1:
	default:
		return fmt.Errorf("node %q matches more than one\n%s", query, strings.Join(matches, "\n"))
	}
	n := byName[matches[0]]
	if n.Addr == "" {
		return fmt.Errorf("node %s has no address", n.Name)
	}
	cluster, err := a.clusterFor(ctx)
	if err != nil {
		return err
	}
	if cluster.Username == "" {
		return fmt.Errorf("no ssh user for context %s; run vssh once", ctx)
	}
	profile := ProfileName(ctx, n.Name)
	err = a.quiet(bin, []string{"show", profile})
	if err == nil && !overwrite {
		return fmt.Errorf("profile %q already exists (pass --overwrite)", profile)
	}
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return err
		}
	}
	port := cluster.Port
	if port <= 0 {
		port = 22
	}
	addArgs := []string{
		"add", profile,
		"--type", "ssh",
		"--host", n.Addr,
		"--user", cluster.Username,
		"--port", strconv.Itoa(port),
		"--group", ctx,
		"--tags", "kube",
	}
	if cluster.IdentityFile != "" {
		addArgs = append(addArgs, "--identity", cluster.IdentityFile)
	}
	return a.run(bin, addArgs)
}

func (a *app) cmdGC() error {
	root, err := a.configRoot()
	if err != nil {
		return err
	}
	n, err := GC(filepath.Join(root, "shells"), a.get("VEEK8S_SHELL"), a.nowTime())
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stderr, "removed %d\n", n)
	return nil
}

func (a *app) nowTime() time.Time {
	if a.now.IsZero() {
		return time.Now()
	}
	return a.now
}

func (a *app) choose(names []string, query, title, prev string) (string, error) {
	if query == "-" {
		if prev == "" {
			return "", fmt.Errorf("no previous %s", title)
		}
		for _, n := range names {
			if n == prev {
				return prev, nil
			}
		}
		return "", fmt.Errorf("previous %s %q not found", title, prev)
	}
	if query == "" {
		return a.pick(title, names)
	}
	m := Match(names, query)
	switch len(m) {
	case 0:
		return "", fmt.Errorf("no %s matching %q\n%s", title, query, strings.Join(names, "\n"))
	case 1:
		return m[0], nil
	default:
		return a.pick(title, m)
	}
}

func (a *app) loadKube() (*Kube, error) {
	home, err := a.home()
	if err != nil {
		return nil, err
	}
	paths, err := KubeconfigPaths(a.get("KUBECONFIG"), home)
	if err != nil {
		return nil, err
	}
	return Merge(paths)
}

func (a *app) shellFile() (path, context string, err error) {
	id := a.get("VEEK8S_SHELL")
	context = a.get("VEEK8S_CONTEXT")
	if id == "" || context == "" {
		return "", "", errors.New("run vkx first")
	}
	if !shellIDPattern.MatchString(id) {
		return "", "", fmt.Errorf("invalid VEEK8S_SHELL %q", id)
	}
	root, err := a.configRoot()
	if err != nil {
		return "", "", err
	}
	path = filepath.Join(root, "shells", id+".yaml")
	if _, err := os.Stat(path); err != nil {
		return "", "", errors.New("run vkx first")
	}
	return path, context, nil
}

func (a *app) configRoot() (string, error) {
	if x := strings.TrimSpace(a.get("XDG_CONFIG_HOME")); x != "" {
		return filepath.Join(x, "veek8s"), nil
	}
	home, err := a.home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "veek8s"), nil
}

func (a *app) loadNodes(kubeconfig, context string) ([]node, error) {
	cfg, err := a.clusterFor(context)
	if err != nil {
		return nil, err
	}
	raw, err := a.kubectlJSON(kubeconfig, "get", "nodes", "-o", "json")
	if err != nil {
		return nil, err
	}
	return parseNodes(raw, cfg.Address)
}

func (a *app) kubectlNames(kubeconfig, kind string) ([]string, error) {
	raw, err := a.kubectlJSON(kubeconfig, "get", kind, "-o", "json")
	if err != nil {
		return nil, err
	}
	names, err := parseNames(raw)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no %s", kind)
	}
	return names, nil
}

func (a *app) kubectlJSON(kubeconfig string, args ...string) ([]byte, error) {
	bin, err := a.look("kubectl")
	if err != nil {
		return nil, errors.New("kubectl not found on PATH")
	}
	full := append([]string{"--kubeconfig", kubeconfig}, args...)
	out, err := a.output(bin, full)
	if err != nil {
		return nil, fmt.Errorf("kubectl: %w", err)
	}
	return out, nil
}

type fileConfig struct {
	Clusters map[string]Cluster `yaml:"clusters"`
}

// Cluster is how to SSH nodes in one context.
type Cluster struct {
	Username     string `yaml:"username"`
	IdentityFile string `yaml:"identityFile,omitempty"`
	Port         int    `yaml:"port,omitempty"`
	Address      string `yaml:"address,omitempty"`
}

func (a *app) clusterFor(context string) (Cluster, error) {
	cfg, err := a.loadFileConfig()
	if err != nil {
		return Cluster{}, err
	}
	c := cfg.Clusters[context]
	if _, err := addressOrder(c.Address); err != nil {
		return Cluster{}, err
	}
	if c.Address == "" {
		c.Address = "internalIP"
	}
	return c, nil
}

func (a *app) ensureCluster(context string) (Cluster, error) {
	c, err := a.clusterFor(context)
	if err != nil {
		return Cluster{}, err
	}
	if c.Username != "" {
		return c, nil
	}
	user, err := a.prompt("ssh user: ")
	if err != nil {
		return Cluster{}, err
	}
	if strings.TrimSpace(user) == "" {
		return Cluster{}, errors.New("ssh user is required")
	}
	ident, err := a.prompt("identity file (empty to skip): ")
	if err != nil {
		return Cluster{}, err
	}
	c.Username = user
	c.IdentityFile = strings.TrimSpace(ident)
	if err := a.saveCluster(context, c); err != nil {
		return Cluster{}, err
	}
	return c, nil
}

func (a *app) loadFileConfig() (fileConfig, error) {
	root, err := a.configRoot()
	if err != nil {
		return fileConfig{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fileConfig{Clusters: map[string]Cluster{}}, nil
		}
		return fileConfig{}, err
	}
	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, err
	}
	if cfg.Clusters == nil {
		cfg.Clusters = map[string]Cluster{}
	}
	return cfg, nil
}

func (a *app) saveCluster(context string, c Cluster) error {
	cfg, err := a.loadFileConfig()
	if err != nil {
		return err
	}
	cfg.Clusters[context] = c
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	root, err := a.configRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return err
	}
	return writePrivate(filepath.Join(root, "config.yaml"), out)
}

func writePrivate(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
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
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	done = true
	return os.Chmod(path, 0o600)
}

func promptLine(prompt string) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("open /dev/tty: %w", err)
	}
	defer tty.Close()
	if _, err := io.WriteString(tty, prompt); err != nil {
		return "", err
	}
	return readLine(tty)
}

func readLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	s, err := br.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	s = strings.TrimRight(s, "\r\n")
	if err != nil && s == "" {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

func outputCmd(name string, args []string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func quietCmd(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func runCmd(name string, args []string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
