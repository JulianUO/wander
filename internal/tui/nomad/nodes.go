package nomad

import (
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/nomad/api"
)

// NodeHost is the Nomad client node host we can reach over SSH.
type NodeHost struct {
	ID, Name, Address, SSHTarget string
}

// ResolveNodeHost looks up a node by ID (full or prefix) and picks an SSH target.
// Preference: Name (often matches ~/.ssh/config Host), then Address from the node
// list stub. Full Node.Info no longer exposes Address in newer nomad/api — we
// fall back to the host part of HTTPAddr if the list stub has no Address.
func ResolveNodeHost(client api.Client, nodeID string) (NodeHost, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return NodeHost{}, fmt.Errorf("empty node id")
	}

	var listAddress string
	node, _, err := client.Nodes().Info(nodeID, nil)
	if err != nil {
		stubs, _, listErr := client.Nodes().List(&api.QueryOptions{Prefix: nodeID})
		if listErr != nil {
			return NodeHost{}, fmt.Errorf("node info: %w", err)
		}
		if len(stubs) == 0 {
			return NodeHost{}, fmt.Errorf("no node found for id %q: %w", nodeID, err)
		}
		if len(stubs) > 1 {
			return NodeHost{}, fmt.Errorf("node id prefix %q matched %d nodes; use a longer prefix", nodeID, len(stubs))
		}
		listAddress = stubs[0].Address
		node, _, err = client.Nodes().Info(stubs[0].ID, nil)
		if err != nil {
			return NodeHost{}, err
		}
	} else {
		listAddress = addressFromNodeList(client, node.ID)
	}

	address := listAddress
	if address == "" {
		address = hostFromHTTPAddr(node.HTTPAddr)
	}

	host := NodeHost{
		ID:      node.ID,
		Name:    node.Name,
		Address: address,
	}
	host.SSHTarget = pickSSHTarget(host.Name, host.Address)
	if host.SSHTarget == "" {
		return host, fmt.Errorf("node %s (%s) has no Name or Address for SSH", shortID(host.ID), host.Name)
	}
	return host, nil
}

func addressFromNodeList(client api.Client, fullID string) string {
	prefix := fullID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	stubs, _, err := client.Nodes().List(&api.QueryOptions{Prefix: prefix})
	if err != nil {
		return ""
	}
	for _, s := range stubs {
		if s.ID == fullID {
			return s.Address
		}
	}
	return ""
}

func hostFromHTTPAddr(httpAddr string) string {
	httpAddr = strings.TrimSpace(httpAddr)
	if httpAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(httpAddr)
	if err != nil {
		// already host-only or non-standard
		return httpAddr
	}
	return host
}

// pickSSHTarget prefers the member IP so we hit the selected host, not a
// ~/.ssh/config Host alias / DNS name that may point at the Nomad API server.
func pickSSHTarget(name, address string) string {
	name = strings.TrimSpace(name)
	address = strings.TrimSpace(address)
	if address != "" {
		return address
	}
	return name
}

func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}
