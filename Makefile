BINARY := menu-easy
BUILD_DIR := bin
PREFIX ?= /usr/local
USER_PREFIX ?= $(HOME)/.local
DESTDIR ?=
GO ?= go
LDFLAGS ?= -s -w

.PHONY: build test install install-user

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -buildvcs=false -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/menu-easy

test:
	$(GO) test -buildvcs=false ./...

install: build
	install -Dm755 $(BUILD_DIR)/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)
	install -Dm644 data/menu-easy.desktop $(DESTDIR)$(PREFIX)/share/applications/menu-easy.desktop
	install -Dm644 data/menu-easy.svg $(DESTDIR)$(PREFIX)/share/icons/hicolor/scalable/apps/menu-easy.svg

install-user: build
	install -Dm755 $(BUILD_DIR)/$(BINARY) $(USER_PREFIX)/bin/$(BINARY)
	install -Dm644 data/menu-easy.desktop $(USER_PREFIX)/share/applications/menu-easy.desktop
	install -Dm644 data/menu-easy.svg $(USER_PREFIX)/share/icons/hicolor/scalable/apps/menu-easy.svg
