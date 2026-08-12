# Menu Easy

Menu Easy is a Linux application launcher written in Go and inspired by
mintMenu. It lists installed applications, supports search, groups entries by
category, and stores favorite applications.

The UI is built with Gio and compiles to a single executable. It does not
depend on GTK, Qt, Electron, Python, or D-Bus services. It only needs the
system libraries required for a native Linux window: X11 or Wayland, EGL, and
xkbcommon.

## Features

- reads `.desktop` files using the freedesktop/XDG conventions;
- localized names, comments, and keywords;
- categories, incremental search, and sorted results;
- persistent favorites in `$XDG_CONFIG_HOME/menu-easy/config.json`;
- PNG/JPEG/SVG/XPM icons from the active desktop/window-manager icon theme,
  including inherited themes and pixmap directories, with a generated fallback;
- safe process launching without passing commands through a shell;
- logout, reboot, and power-off buttons in the footer;
- support for `Terminal=true` applications;
- closes on `Esc`, focus loss, or after launching an application;
- native Xorg and Wayland backends in the same binary.

## Build

You need Go 1.24 or newer, a C compiler, and the platform library headers.

On Debian, Ubuntu, and Linux Mint:

```sh
sudo apt install gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev \
  libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \
  libxcursor-dev libvulkan-dev
make build
```

The binary is written to `bin/menu-easy`. To run the tests:

```sh
make test
```

## User Install

```sh
make install-user
~/.local/bin/menu-easy
```

Make sure `~/.local/bin` is in the `PATH` used by your graphical session. To
install under `/usr/local` instead:

```sh
sudo make install
```

## IceWM

To add the launcher button to the taskbar, add this line to
`$XDG_CONFIG_HOME/icewm/toolbar` or `~/.icewm/toolbar`:

```text
prog "Menu Easy" menu-easy menu-easy
```

For a global shortcut, add this line to `$XDG_CONFIG_HOME/icewm/keys` or
`~/.icewm/keys`:

```text
key "Super+space" menu-easy
```

If `menu-easy` is not in IceWM's `PATH`, use the full path:
`/home/YOUR_USER/.local/bin/menu-easy`. Reload the configuration without
restarting the session:

```sh
icesh toolbar
icesh keys
```

## Xorg and Wayland

The default binary contains both backends and selects the current session type
at startup. On Wayland, the compositor decides the initial window position: the
base protocol does not allow a portable application to anchor itself to
absolute panel coordinates. All other features are the same.

To deliberately build a binary with only one backend:

```sh
go build -tags nox11 -o menu-easy-wayland ./cmd/menu-easy
go build -tags nowayland -o menu-easy-x11 ./cmd/menu-easy
```
