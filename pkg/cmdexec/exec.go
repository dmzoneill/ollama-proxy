// Package cmdexec starts allowlisted external programs without a shell.
package cmdexec

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// allowed is the set of executables this proxy is willing to spawn.
// Names are matched on the base path so a configured absolute path such as
// /usr/bin/pactl is accepted only when the program itself is allowlisted.
var allowed = map[string]struct{}{
	"gsettings":   {},
	"kill":        {},
	"modprobe":    {},
	"pacat":       {},
	"parec":       {},
	"pactl":       {},
	"piper":       {},
	"pw-cli":      {},
	"pw-loopback": {},
	"python3":     {},
	"v4l2-ctl":    {},
	"whisper-cpp": {},
}

// CommandContext validates name and args, then returns a command that runs
// them directly (arguments are not interpreted by a shell).
func CommandContext(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	if err := validate(name, args); err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, name, args...), nil //nolint:gosec // G204: name is allowlisted and args are not passed through a shell
}

// Command is CommandContext bound to context.Background.
func Command(name string, args ...string) (*exec.Cmd, error) {
	return CommandContext(context.Background(), name, args...)
}

func validate(name string, args []string) error {
	base := filepath.Base(name)
	if _, ok := allowed[base]; !ok {
		return fmt.Errorf("refusing to execute %q", name)
	}
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("executable path contains NUL")
	}
	for _, arg := range args {
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("argument contains NUL")
		}
	}
	return nil
}
