package session

import (
	"errors"
	"testing"
)

func TestLogoutCommandOrdersDesktopFirst(t *testing.T) {
	cases := []struct {
		desktop string
		first   string
	}{
		{"X-Cinnamon", "cinnamon-session-quit"},
		{"MATE", "mate-session-save"},
		{"ubuntu:GNOME", "gnome-session-quit"},
		{"XFCE", "xfce4-session-logout"},
		{"", "cinnamon-session-quit"},
	}
	for _, c := range cases {
		commands := logoutCommands(c.desktop, "c20")
		if got := commands[0][0]; got != c.first {
			t.Errorf("logoutCommands(%q)[0] = %q, want %q", c.desktop, got, c.first)
		}
		if last := commands[len(commands)-1]; last[0] != "loginctl" || last[2] != "c20" {
			t.Errorf("logoutCommands(%q) last = %#v, want loginctl terminate-session c20", c.desktop, last)
		}
	}
}

func TestLogoutCommandDropsLoginctlWithoutSessionID(t *testing.T) {
	commands := logoutCommands("", "")
	for _, args := range commands {
		if args[0] == "loginctl" {
			t.Fatal("loginctl fallback should be absent without a session id")
		}
	}
}

func TestRunFirstPicksAvailableCommand(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "true" {
			return "/usr/bin/true", nil
		}
		return "", errors.New("not found")
	}
	err := runFirst([][]string{
		{"missing-command"},
		{"true"},
	}, lookup)
	if err != nil {
		t.Fatalf("runFirst() error = %v, want nil", err)
	}
}

func TestRunFirstReportsNothingFound(t *testing.T) {
	lookup := func(string) (string, error) {
		return "", errors.New("not found")
	}
	if err := runFirst([][]string{{"missing-command"}}, lookup); err == nil {
		t.Fatal("runFirst() error = nil, want an error when no command is available")
	}
}
