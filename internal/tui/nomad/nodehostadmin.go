package nomad

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// HostAdminAction is a node-host menu action (separate from alloc/job AdminAction).
type HostAdminAction int8

// Order = menu sort order. Operational actions first; interactive SSH last.
const (
	HostCheckConsulConfig HostAdminAction = iota
	HostCheckNomadConfig
	HostConsulReload
	HostConsulRestart
	HostNomadRestart
	HostCopySSHCommand
	HostOpenSSH
)

// HostCommandConfig holds configurable remote command templates (placeholders by default).
type HostCommandConfig struct {
	ConsulConfig  string
	NomadConfig   string
	ConsulReload  string
	ConsulRestart string
	NomadRestart  string
}

// DefaultHostCommandConfig returns fleet defaults; override via ~/.wander.yaml if needed.
func DefaultHostCommandConfig() HostCommandConfig {
	return HostCommandConfig{
		ConsulConfig:  "sudo cat /etc/consul.d/consul.hcl",
		NomadConfig:   "sudo cat /etc/nomad.d/nomad.hcl",
		ConsulReload:  "sudo consul reload",
		ConsulRestart: "sudo systemctl restart consul",
		NomadRestart:  "sudo systemctl restart nomad",
	}
}

var HostAdminActions = map[HostAdminAction]string{
	HostCheckConsulConfig: "Check Consul config",
	HostCheckNomadConfig:  "Check Nomad config",
	HostConsulReload:      "Consul reload",
	HostConsulRestart:     "Consul restart (systemctl)",
	HostNomadRestart:      "Nomad restart (systemctl)",
	HostCopySSHCommand:    "Copy SSH command (second terminal)",
	HostOpenSSH:           "Open SSH in this terminal",
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
	case HostOpenSSH:
		return "host-open-ssh"
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
	case "host-open-ssh":
		return HostOpenSSH
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
	return HostAdminActions[a]
}

func HostAdminHeader(host NodeHost) string {
	return fmt.Sprintf("Host actions │ %s │ IP %s │ ssh %s",
		host.Name, host.Address, host.SSHTarget)
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
	SSHCmd   string
	CopiedOK bool
}

// HostSSHSessionDoneMsg is sent after an interactive ssh session exits.
type HostSSHSessionDoneMsg struct {
	Err  error
	Host NodeHost
}

func GetCmdForHostAdminAction(
	a HostAdminAction,
	host NodeHost,
	cfg HostCommandConfig,
	copyFn func(string) error,
) tea.Cmd {
	return func() tea.Msg {
		if a == HostOpenSSH {
			return HostAdminActionCompleteMsg{Action: a, Host: host, SSHCmd: FormatSSHConnectCommand(host.SSHTarget)}
		}
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
		header := fmt.Sprintf("$ ssh %s -- %s\n\n", host.SSHTarget, remote)
		return HostAdminActionCompleteMsg{
			Err:    err,
			Action: a,
			Host:   host,
			Output: header + out,
		}
	}
}
