package veek8s

import (
	"encoding/json"
	"fmt"
	"strings"
)

type node struct {
	Name string
	Addr string
}

type k8sList struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Addresses []struct {
				Type    string `json:"type"`
				Address string `json:"address"`
			} `json:"addresses"`
		} `json:"status"`
	} `json:"items"`
}

func parseNodes(raw []byte, prefer string) ([]node, error) {
	order, err := addressOrder(prefer)
	if err != nil {
		return nil, err
	}
	var list k8sList
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("nodes: %w", err)
	}
	out := make([]node, 0, len(list.Items))
	for _, item := range list.Items {
		if item.Metadata.Name == "" {
			continue
		}
		addrs := map[string]string{}
		for _, a := range item.Status.Addresses {
			if _, ok := addrs[a.Type]; ok {
				continue
			}
			addrs[a.Type] = a.Address
		}
		out = append(out, node{Name: item.Metadata.Name, Addr: firstAddr(addrs, order)})
	}
	return out, nil
}

func parseNames(raw []byte) ([]string, error) {
	var list k8sList
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	names := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		if item.Metadata.Name != "" {
			names = append(names, item.Metadata.Name)
		}
	}
	return names, nil
}

func addressOrder(prefer string) ([]string, error) {
	switch prefer {
	case "", "internalIP":
		return []string{"InternalIP", "Hostname", "ExternalIP"}, nil
	case "hostname":
		return []string{"Hostname", "InternalIP", "ExternalIP"}, nil
	case "externalIP":
		return []string{"ExternalIP", "InternalIP", "Hostname"}, nil
	default:
		return nil, fmt.Errorf("address %q must be internalIP, hostname, or externalIP", prefer)
	}
}

func firstAddr(addrs map[string]string, order []string) string {
	for _, k := range order {
		if addrs[k] != "" && !strings.HasPrefix(addrs[k], "-") {
			return addrs[k]
		}
	}
	return ""
}
