package veek8s

import (
	"strings"
	"testing"
)

const nodeJSON = `{
  "items": [
    {"metadata": {"name": ""}, "status": {"addresses": [{"type": "InternalIP", "address": "10.0.0.1"}]}},
    {"metadata": {"name": "n1"}, "status": {"addresses": [
      {"type": "InternalIP", "address": "10.0.0.2"},
      {"type": "InternalIP", "address": "10.9.9.9"},
      {"type": "Hostname", "address": "n1.example"},
      {"type": "ExternalIP", "address": "203.0.113.5"}
    ]}},
    {"metadata": {"name": "n2"}, "status": {"addresses": [
      {"type": "Hostname", "address": "n2.example"},
      {"type": "ExternalIP", "address": "-bad"}
    ]}},
    {"metadata": {"name": "n3"}, "status": {"addresses": [
      {"type": "InternalIP", "address": "-flag"}
    ]}}
  ]
}`

func TestParseNodes(t *testing.T) {
	got, err := parseNodes([]byte(nodeJSON), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Name != "n1" || got[0].Addr != "10.0.0.2" {
		t.Fatalf("default %+v", got)
	}
	if got[1].Addr != "n2.example" {
		t.Fatalf("fallback %+v", got[1])
	}
	if got[2].Addr != "" {
		t.Fatalf("flag address accepted %+v", got[2])
	}
	host, err := parseNodes([]byte(nodeJSON), "hostname")
	if err != nil || host[0].Addr != "n1.example" {
		t.Fatalf("hostname %+v %v", host, err)
	}
	ext, err := parseNodes([]byte(nodeJSON), "externalIP")
	if err != nil || ext[0].Addr != "203.0.113.5" || ext[1].Addr != "n2.example" {
		t.Fatalf("external %+v %v", ext, err)
	}
	if _, err := parseNodes([]byte(nodeJSON), "nope"); err == nil {
		t.Fatal("bad prefer")
	}
	if _, err := parseNodes([]byte("{"), ""); err == nil {
		t.Fatal("bad json")
	}
	names, err := parseNames([]byte(`{"items":[{"metadata":{"name":"kube-system"}},{"metadata":{"name":""}},{"metadata":{"name":"default"}}]}`))
	if err != nil || strings.Join(names, ",") != "kube-system,default" {
		t.Fatalf("names %+v %v", names, err)
	}
	if _, err := parseNames([]byte("{")); err == nil {
		t.Fatal("bad name json")
	}
}
