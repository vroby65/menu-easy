package launch

import (
	"errors"
	"reflect"
	"testing"
)

func TestTerminalCommand(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "foot" {
			return "/usr/bin/foot", nil
		}
		return "", errors.New("not found")
	}
	got, err := terminalCommand([]string{"htop", "--readonly"}, "", lookup)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/usr/bin/foot", "-e", "htop", "--readonly"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("terminalCommand()=%#v, want %#v", got, want)
	}
}

func TestTerminalCommandUsesEnvironment(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "my-terminal" {
			return "/opt/bin/my-terminal", nil
		}
		return "", errors.New("not found")
	}
	got, err := terminalCommand([]string{"htop"}, "my-terminal", lookup)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/opt/bin/my-terminal", "-e", "htop"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("terminalCommand()=%#v, want %#v", got, want)
	}
}
