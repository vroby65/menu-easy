// Package session performs desktop session actions: logout, reboot and power
// off. Commands are chosen by availability and launched detached, so the menu
// can close as soon as the request is sent.
package session

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Logout ends the current graphical session using the running desktop's
// session manager, falling back to the other common managers and finally to
// loginctl.
func Logout() error {
	return runFirst(logoutCommands(currentDesktop(), os.Getenv("XDG_SESSION_ID")), exec.LookPath)
}

// Reboot restarts the machine through logind, which prompts for authorization
// when needed. systemctl and shutdown are fallbacks for non-systemd systems.
func Reboot() error {
	commands := [][]string{
		{"dbus-send", "--system", "--print-reply", "--dest=org.freedesktop.login1", "/org/freedesktop/login1", "org.freedesktop.login1.Manager.Reboot", "boolean:true"},
		{"systemctl", "reboot"},
		{"shutdown", "-r", "now"},
	}
	return runFirst(commands, exec.LookPath)
}

// PowerOff shuts the machine down through logind, with systemctl and shutdown
// as fallbacks.
func PowerOff() error {
	commands := [][]string{
		{"dbus-send", "--system", "--print-reply", "--dest=org.freedesktop.login1", "/org/freedesktop/login1", "org.freedesktop.login1.Manager.PowerOff", "boolean:true"},
		{"systemctl", "poweroff"},
		{"shutdown", "-h", "now"},
	}
	return runFirst(commands, exec.LookPath)
}

// logoutCommands lists session-manager commands in priority order, moving the
// manager of the running desktop first so a system with several managers
// installed still logs out of the active session.
func logoutCommands(desktop, sessionID string) [][]string {
	commands := [][]string{
		{"cinnamon-session-quit", "--logout"},
		{"mate-session-save", "--logout"},
		{"gnome-session-quit", "--logout"},
		{"xfce4-session-logout", "--logout"},
	}
	if sessionID != "" {
		commands = append(commands, []string{"loginctl", "terminate-session", sessionID})
	}
	switch {
	case strings.Contains(strings.ToLower(desktop), "cinnamon"):
	case strings.Contains(strings.ToLower(desktop), "mate"):
		commands[0], commands[1] = commands[1], commands[0]
	case strings.Contains(strings.ToLower(desktop), "gnome"), strings.Contains(strings.ToLower(desktop), "ubuntu"):
		commands[0], commands[2] = commands[2], commands[0]
	case strings.Contains(strings.ToLower(desktop), "xfce"):
		commands[0], commands[3] = commands[3], commands[0]
	case strings.Contains(strings.ToLower(desktop), "icewm"):
		commands = append([][]string{{"icesh", "logout"}}, commands...)
	}
	return commands
}

func currentDesktop() string {
	return strings.Join([]string{
		os.Getenv("XDG_CURRENT_DESKTOP"),
		os.Getenv("DESKTOP_SESSION"),
		os.Getenv("GDMSESSION"),
		os.Getenv("WINDOWMANAGER"),
	}, ":")
}

// runFirst starts the first command whose executable is available and returns
// its error. The process is released instead of waited on because the session
// ends asynchronously.
func runFirst(commands [][]string, lookup func(string) (string, error)) error {
	for _, args := range commands {
		if len(args) == 0 {
			continue
		}
		if _, err := lookup(args[0]); err != nil {
			continue
		}
		return runDetached(args)
	}
	return errors.New("nessun comando di sessione disponibile")
}

func runDetached(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
