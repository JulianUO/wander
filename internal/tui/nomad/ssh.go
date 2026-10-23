package nomad

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultSSHTimeout = 60 * time.Second

// RunSSHCommand runs `ssh <target> -- <remoteCmd>` using the system OpenSSH client
// so ~/.ssh/config (Host, ProxyJump, IdentityFile, User) applies.
// No Wander-specific -o overrides are added.
func RunSSHCommand(sshTarget, remoteCmd string) (string, error) {
	sshTarget = strings.TrimSpace(sshTarget)
	remoteCmd = strings.TrimSpace(remoteCmd)
	if sshTarget == "" {
		return "", fmt.Errorf("empty ssh target")
	}
	if remoteCmd == "" {
		return "", fmt.Errorf("empty remote command")
	}
	if strings.Contains(remoteCmd, "PLACEHOLDER") {
		return "", fmt.Errorf(
			"remote command still contains PLACEHOLDER — set wander_host_cmd_* in ~/.wander.yaml (see README)",
		)
	}

	ctxCmd := exec.Command("ssh", sshTarget, "--", remoteCmd)
	var stdout, stderr bytes.Buffer
	ctxCmd.Stdout = &stdout
	ctxCmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- ctxCmd.Run() }()

	select {
	case err := <-done:
		out := strings.TrimSpace(stdout.String())
		errOut := strings.TrimSpace(stderr.String())
		combined := out
		if errOut != "" {
			if combined != "" {
				combined += "\n\n"
			}
			combined += "[stderr]\n" + errOut
		}
		if err != nil {
			if combined == "" {
				combined = err.Error()
			}
			return combined, fmt.Errorf("ssh %s: %w", sshTarget, err)
		}
		return combined, nil
	case <-time.After(defaultSSHTimeout):
		_ = ctxCmd.Process.Kill()
		return "", fmt.Errorf("ssh to %s timed out after %s", sshTarget, defaultSSHTimeout)
	}
}

// FormatSSHConnectCommand returns a copy-pasteable interactive ssh invocation.
func FormatSSHConnectCommand(sshTarget string) string {
	return fmt.Sprintf("ssh %s", strings.TrimSpace(sshTarget))
}
