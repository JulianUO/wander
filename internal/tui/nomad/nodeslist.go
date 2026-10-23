package nomad

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/nomad/api"
	"github.com/robinovitch61/wander/internal/tui/components/page"
	"github.com/robinovitch61/wander/internal/tui/formatter"
	"github.com/robinovitch61/wander/internal/tui/message"
)

const (
	HostKindClient = "client"
	HostKindServer = "server"
)

// HostListEntry is one row in the Nodes & Members list.
type HostListEntry struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Status     string `json:"status"`
	Datacenter string `json:"datacenter"`
	Region     string `json:"region"`
}

func (e HostListEntry) ToNodeHost() NodeHost {
	h := NodeHost{
		ID:      e.ID,
		Name:    e.Name,
		Address: e.Address,
	}
	h.SSHTarget = pickSSHTarget(e.Name, e.Address)
	return h
}

func HostListEntryFromKey(key string) (HostListEntry, error) {
	var e HostListEntry
	if err := json.Unmarshal([]byte(key), &e); err != nil {
		return HostListEntry{}, err
	}
	return e, nil
}

func toHostListKey(e HostListEntry) string {
	b, err := json.Marshal(e)
	if err != nil {
		return e.Kind + keySeparator + e.Name + keySeparator + e.Address
	}
	return string(b)
}

// FetchNodesAndMembers lists Nomad client nodes and server members for SSH targeting.
func FetchNodesAndMembers(client api.Client) tea.Cmd {
	return func() tea.Msg {
		var entries []HostListEntry

		stubs, _, err := client.Nodes().List(nil)
		if err != nil {
			return message.ErrMsg{Err: fmt.Errorf("list nodes: %w", err)}
		}
		for _, s := range stubs {
			entries = append(entries, HostListEntry{
				Kind:       HostKindClient,
				ID:         s.ID,
				Name:       s.Name,
				Address:    s.Address,
				Status:     s.Status,
				Datacenter: s.Datacenter,
			})
		}

		members, memErr := client.Agent().Members()
		if memErr != nil {
			// Nodes still usable without agent members (ACL / token scope).
			entries = append(entries, HostListEntry{
				Kind:    "warning",
				Name:    "(server members unavailable)",
				Address: memErr.Error(),
				Status:  "n/a",
			})
		} else if members != nil {
			for _, m := range members.Members {
				if m == nil {
					continue
				}
				role := m.Tags["role"]
				if role == "" {
					role = m.Tags["mode"]
				}
				// Prefer serf members that look like servers; still show others tagged oddly.
				kind := HostKindServer
				if role == "client" {
					// Client often also appears under /v1/nodes; skip duplicate by name+addr if present.
					if hostListHas(entries, m.Name, m.Addr) {
						continue
					}
					kind = HostKindClient
				}
				entries = append(entries, HostListEntry{
					Kind:       kind,
					ID:         m.Name,
					Name:       m.Name,
					Address:    m.Addr,
					Status:     m.Status,
					Datacenter: m.Tags["dc"],
					Region:     m.Tags["region"],
				})
			}
		}

		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Kind != entries[j].Kind {
				// servers first, then clients, warnings last
				order := map[string]int{HostKindServer: 0, HostKindClient: 1, "warning": 2}
				return order[entries[i].Kind] < order[entries[j].Kind]
			}
			if entries[i].Name == entries[j].Name {
				return entries[i].Address < entries[j].Address
			}
			return entries[i].Name < entries[j].Name
		})

		header, rows := hostListAsTable(entries)
		return PageLoadedMsg{Page: NodesPage, TableHeader: header, AllPageRows: rows}
	}
}

func hostListHas(entries []HostListEntry, name, addr string) bool {
	for _, e := range entries {
		if e.Name == name || (addr != "" && e.Address == addr) {
			return true
		}
	}
	return false
}

func hostListAsTable(entries []HostListEntry) ([]string, []page.Row) {
	columns := []string{"Kind", "Name", "Address", "Status", "DC", "Region", "Node ID"}
	var content [][]string
	var keys []string
	for _, e := range entries {
		id := e.ID
		if e.Kind == HostKindClient {
			id = formatter.ShortAllocID(e.ID)
		}
		content = append(content, []string{
			e.Kind,
			e.Name,
			e.Address,
			e.Status,
			emptyDash(e.Datacenter),
			emptyDash(e.Region),
			emptyDash(id),
		})
		keys = append(keys, toHostListKey(e))
	}
	table := formatter.GetRenderedTableAsString(columns, content)
	var rows []page.Row
	for i, row := range table.ContentRows {
		rows = append(rows, page.Row{Key: keys[i], Row: row})
	}
	return table.HeaderRows, rows
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
