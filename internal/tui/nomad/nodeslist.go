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

// HostListEntry is one row in the cluster members list (name + IP).
type HostListEntry struct {
	Kind    string `json:"kind"` // client | server
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"` // IP
	Status  string `json:"status"`
}

func (e HostListEntry) ToNodeHost() NodeHost {
	h := NodeHost{
		ID:      e.ID,
		Name:    e.Name,
		Address: e.Address,
	}
	// Prefer Name so ~/.ssh/config Host entries match; fall back to IP.
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
		return e.Name + keySeparator + e.Address
	}
	return string(b)
}

// FetchNodesAndMembers builds a de-duplicated cluster member list (name + IP)
// from Nomad client nodes and agent/server members.
func FetchNodesAndMembers(client api.Client) tea.Cmd {
	return func() tea.Msg {
		byIP := map[string]HostListEntry{}

		stubs, _, err := client.Nodes().List(nil)
		if err != nil {
			return message.ErrMsg{Err: fmt.Errorf("list nodes: %w", err)}
		}
		for _, s := range stubs {
			ip := strings.TrimSpace(s.Address)
			if ip == "" {
				continue
			}
			byIP[ip] = HostListEntry{
				Kind:    HostKindClient,
				ID:      s.ID,
				Name:    s.Name,
				Address: ip,
				Status:  s.Status,
			}
		}

		members, memErr := client.Agent().Members()
		if memErr == nil && members != nil {
			for _, m := range members.Members {
				if m == nil {
					continue
				}
				ip := strings.TrimSpace(m.Addr)
				if ip == "" {
					continue
				}
				role := m.Tags["role"]
				if role == "" {
					role = m.Tags["mode"]
				}
				kind := HostKindServer
				if role == "client" {
					kind = HostKindClient
				}
				existing, ok := byIP[ip]
				name := m.Name
				if ok && existing.Name != "" {
					// Keep node registry name when present (often cleaner).
					name = existing.Name
				}
				id := existing.ID
				if id == "" {
					id = m.Name
				}
				if ok && kind == HostKindClient && existing.Kind == HostKindClient {
					kind = HostKindClient
				}
				// If both client and server seen on same IP, prefer server label only when role says so.
				if ok && existing.Kind == HostKindServer {
					kind = HostKindServer
				}
				if kind == HostKindServer || !ok {
					byIP[ip] = HostListEntry{
						Kind:    kind,
						ID:      id,
						Name:    name,
						Address: ip,
						Status:  m.Status,
					}
				} else if ok {
					e := existing
					if e.Status == "" {
						e.Status = m.Status
					}
					byIP[ip] = e
				}
			}
		}

		entries := make([]HostListEntry, 0, len(byIP))
		for _, e := range byIP {
			entries = append(entries, e)
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Kind != entries[j].Kind {
				if entries[i].Kind == HostKindServer {
					return true
				}
				if entries[j].Kind == HostKindServer {
					return false
				}
			}
			return entries[i].Name < entries[j].Name
		})

		if memErr != nil && len(entries) == 0 {
			return message.ErrMsg{Err: fmt.Errorf("list members: %w", memErr)}
		}

		header, rows := hostListAsTable(entries)
		return PageLoadedMsg{Page: NodesPage, TableHeader: header, AllPageRows: rows}
	}
}

func hostListAsTable(entries []HostListEntry) ([]string, []page.Row) {
	columns := []string{"Name", "IP", "Role", "Status"}
	var content [][]string
	var keys []string
	for _, e := range entries {
		content = append(content, []string{
			e.Name,
			e.Address,
			e.Kind,
			emptyDash(e.Status),
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
