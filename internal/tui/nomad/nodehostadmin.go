package nomad

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// HostAdminAction is a node-host menu action (separate from alloc/job AdminAction).
type HostAdminAction int8

const (
	HostCheckConsulConfig HostAdminAction = iota
	HostCheckNomadConfig
	HostConsulReload
	HostConsulRestart
	HostNomadRestart
	HostCopySSHCommand
)

// HostCommandConfig holds configurable remote command templates (placeholders by default).
type HostCommandConfig struct {
	ConsulConfig  string
	NomadConfig   string
	ConsulReload  string
	ConsulRestart string
	NomadRestart  string
}

// DefaultHostCommandConfig returns PLACEHOLDER commands — replace via ~/.wander.yaml.
func DefaultHostCommandConfig() HostCommandConfig {
	return HostCommandConfig{
		ConsulConfig:  "PLACEHOLDER: show consul config (e.g. sudo cat /etc/consul.d/consul.hcl)",
		NomadConfig:   "PLACEHOLDER: show nomad config (e.g. sudo cat /etc/nomad.d/nomad.hcl)",
		ConsulReload:  "PLACEHOLDER: reload consul (e.g. sudo systemctl reload consul)",
		ConsulRestart: "PLACEHOLDER: restart consul (e.g. sudo systemctl restart consul)",
		NomadRestart:  "PLACEHOLDER: restart nomad client (e.g. sudo systemctl restart nomad)",
	}
}

var HostAdminActions = map[HostAdminAction]string{
	HostCheckConsulConfig: "Check Consul config",
	HostCheckNomadConfig:  "Check Nomad config",
	HostConsulReload:      "Consul reload",
	HostConsulRestart:     "Consul restart",
	HostNomadRestart:      "Nomad restart",
	HostCopySSHCommand:    "Copy SSH command",
}

func HostAdminActionToKey(a HostAdminAction) string {
	switch a {
	case HostCheckConsulConfig:
		return "host-check-consul-config"
	case HostCheckNomadConfig:
		return "host-check-nomad-config"
	case HostConsulReload:
		return "host-consul-reload"
	case HostConsulRestart:
		return "host-consul-restart"
	case HostNomadRestart:
		return "host-nomad-restart"
	case HostCopySSHCommand:
		return "host-copy-ssh"
	default:
		return ""
	}
}

func KeyToHostAdminAction(key string) HostAdminAction {
	switch key {
	case "host-check-consul-config":
		return HostCheckConsulConfig
	case "host-check-nomad-config":
		return HostCheckNomadConfig
	case "host-consul-reload":
		return HostConsulReload
	case "host-consul-restart":
		return HostConsulRestart
	case "host-nomad-restart":
		return HostNomadRestart
	case "host-copy-ssh":
		return HostCopySSHCommand
	default:
		return -1
	}
}

func HostAdminActionNeedsConfirm(a HostAdminAction) bool {
	switch a {
	case HostConsulReload, HostConsulRestart, HostNomadRestart:
		return true
	default:
		return false
	}
}

func GetHostAdminText(a HostAdminAction, host NodeHost) string {
	label := HostAdminActions[a]
	target := host.SSHTarget
	if target == "" {
		target = host.Name
	}
	return fmt.Sprintf("%s on %s (%s)", label, target, shortID(host.ID))
}

func remoteCmdFor(a HostAdminAction, cfg HostCommandConfig) string {
	switch a {
	case HostCheckConsulConfig:
		return cfg.ConsulConfig
	case HostCheckNomadConfig:
		return cfg.NomadConfig
	case HostConsulReload:
		return cfg.ConsulReload
	case HostConsulRestart:
		return cfg.ConsulRestart
	case HostNomadRestart:
		return cfg.NomadRestart
	default:
		return ""
	}
}

type HostAdminActionCompleteMsg struct {
	Err      error
	Action   HostAdminAction
	Host     NodeHost
	Output   string
	SSHCmd   string // set for CopySSHCommand
	CopiedOK bool
}

func GetCmdForHostAdminAction(
	a HostAdminAction,
	host NodeHost,
	cfg HostCommandConfig,
	copyFn func(string) error,
) tea.Cmd {
	return func() tea.Msg {
		if a == HostCopySSHCommand {
			cmd := FormatSSHConnectCommand(host.SSHTarget)
			err := copyFn(cmd)
			return HostAdminActionCompleteMsg{
				Err:      err,
				Action:   a,
				Host:     host,
				SSHCmd:   cmd,
				CopiedOK: err == nil,
				Output:   cmd,
			}
		}

		remote := remoteCmdFor(a, cfg)
		out, err := RunSSHCommand(host.SSHTarget, remote)
		if out == "" && err != nil {
			out = err.Error()
		} else if err != nil && !strings.Contains(out, err.Error()) {
			out = strings.TrimSpace(out + "\n" + err.Error())
		}
		return HostAdminActionCompleteMsg{
			Err:    err,
			Action: a,
			Host:   host,
			Output: out,
		}
	}
}
