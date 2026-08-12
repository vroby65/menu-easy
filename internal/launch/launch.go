package launch

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"menu-easy/internal/desktop"
)

type terminal struct {
	name   string
	prefix []string
}

var terminals = []terminal{
	{name: "x-terminal-emulator", prefix: []string{"-e"}},
	{name: "foot", prefix: []string{"-e"}},
	{name: "xterm", prefix: []string{"-e"}},
	{name: "alacritty", prefix: []string{"-e"}},
	{name: "kitty"},
	{name: "wezterm", prefix: []string{"start", "--"}},
	{name: "konsole", prefix: []string{"-e"}},
	{name: "gnome-terminal", prefix: []string{"--"}},
	{name: "xfce4-terminal", prefix: []string{"-x"}},
}

// Start launches an application detached from the menu process.
func Start(entry desktop.Entry) error {
	args, err := entry.CommandArgs()
	if err != nil {
		return err
	}
	if entry.Terminal {
		args, err = terminalCommand(args, os.Getenv("TERMINAL"), exec.LookPath)
		if err != nil {
			return err
		}
	}
	cmd := exec.Command(args[0], args[1:]...)
	if entry.Path != "" {
		if info, err := os.Stat(entry.Path); err == nil && info.IsDir() {
			cmd.Dir = entry.Path
		}
	}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func terminalCommand(command []string, preferred string, lookup func(string) (string, error)) ([]string, error) {
	if preferred != "" {
		fields := strings.Fields(preferred)
		if len(fields) > 0 {
			path, err := lookup(fields[0])
			if err == nil {
				args := append([]string{path}, fields[1:]...)
				args = append(args, "-e")
				return append(args, command...), nil
			}
		}
	}
	for _, candidate := range terminals {
		path, err := lookup(candidate.name)
		if err != nil {
			continue
		}
		args := append([]string{path}, candidate.prefix...)
		return append(args, command...), nil
	}
	return nil, errors.New("nessun emulatore di terminale disponibile")
}
