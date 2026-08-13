//go:build linux && !android

package windowctrl

/*
#cgo LDFLAGS: -lX11
#include <stdlib.h>
#include <string.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

typedef struct {
	unsigned long flags;
	unsigned long functions;
	unsigned long decorations;
	long input_mode;
	unsigned long status;
} MotifWmHints;

static void setNoDecorations(Display *display, Window window) {
	Atom property = XInternAtom(display, "_MOTIF_WM_HINTS", False);
	MotifWmHints hints;
	hints.flags = 2;
	hints.functions = 0;
	hints.decorations = 0;
	hints.input_mode = 0;
	hints.status = 0;
	XChangeProperty(display, window, property, property, 32, PropModeReplace, (unsigned char *)&hints, 5);
}

static void setAtomProperty(Display *display, Window window, const char *propertyName, const char *atomName) {
	Atom property = XInternAtom(display, propertyName, False);
	Atom atom = XInternAtom(display, atomName, False);
	XChangeProperty(display, window, property, XA_ATOM, 32, PropModeReplace, (unsigned char *)&atom, 1);
}

static void addNetWMState(Display *display, Window window, const char *atomName) {
	Atom state = XInternAtom(display, "_NET_WM_STATE", False);
	Atom atom = XInternAtom(display, atomName, False);
	XEvent event;
	memset(&event, 0, sizeof(event));
	event.xclient.type = ClientMessage;
	event.xclient.display = display;
	event.xclient.window = window;
	event.xclient.message_type = state;
	event.xclient.format = 32;
	event.xclient.data.l[0] = 1;
	event.xclient.data.l[1] = atom;
	event.xclient.data.l[3] = 1;
	XSendEvent(display, XDefaultRootWindow(display), False,
		SubstructureRedirectMask | SubstructureNotifyMask, &event);
}

static void setMenuWindowState(Display *display, Window window) {
	Atom property = XInternAtom(display, "_NET_WM_STATE", False);
	Atom atoms[2];
	atoms[0] = XInternAtom(display, "_NET_WM_STATE_SKIP_TASKBAR", False);
	atoms[1] = XInternAtom(display, "_NET_WM_STATE_SKIP_PAGER", False);
	XChangeProperty(display, window, property, XA_ATOM, 32, PropModeReplace, (unsigned char *)atoms, 2);
	addNetWMState(display, window, "_NET_WM_STATE_SKIP_TASKBAR");
	addNetWMState(display, window, "_NET_WM_STATE_SKIP_PAGER");
}

static void setWindowBackground(Display *display, Window window, unsigned char red, unsigned char green, unsigned char blue) {
	XColor color;
	color.red = (unsigned short)red * 257;
	color.green = (unsigned short)green * 257;
	color.blue = (unsigned short)blue * 257;
	color.flags = DoRed | DoGreen | DoBlue;
	if (XAllocColor(display, XDefaultColormap(display, XDefaultScreen(display)), &color)) {
		XSetWindowBackground(display, window, color.pixel);
		XClearWindow(display, window);
	}
}

static int getWindowSize(Display *display, Window window, int *width, int *height) {
	XWindowAttributes attrs;
	if (!XGetWindowAttributes(display, window, &attrs) || attrs.width <= 0 || attrs.height <= 0) {
		return 0;
	}
	*width = attrs.width;
	*height = attrs.height;
	return 1;
}

static int getCardinalProperty(Display *display, Window window, const char *name, unsigned long *out, int max) {
	Atom property = XInternAtom(display, name, True);
	Atom actualType;
	int actualFormat;
	unsigned long nitems;
	unsigned long bytesAfter;
	unsigned char *data = 0;
	int count = 0;
	int status;
	if (property == None) {
		return 0;
	}
	status = XGetWindowProperty(display, window, property, 0, max, False, XA_CARDINAL,
		&actualType, &actualFormat, &nitems, &bytesAfter, &data);
	if (status != Success || data == 0) {
		return 0;
	}
	if (actualType == XA_CARDINAL && actualFormat == 32) {
		unsigned long *values = (unsigned long *)data;
		count = nitems < (unsigned long)max ? (int)nitems : max;
		for (int i = 0; i < count; i++) {
			out[i] = values[i];
		}
	}
	XFree(data);
	return count;
}
*/
import "C"

import (
	"image"
	"image/color"
	"os/exec"
	"strconv"
	"strings"
	"unsafe"

	"gioui.org/app"
)

type Controller struct {
	display unsafe.Pointer
	window  uintptr
	size    image.Point
	bg      color.NRGBA
	placed  bool
}

func (c *Controller) HandleEvent(event any) {
	if event, ok := event.(app.X11ViewEvent); ok {
		c.SetX11View(event)
	}
}

func (c *Controller) SetX11View(event app.X11ViewEvent) {
	if event.Display == nil || event.Window == 0 {
		return
	}
	c.display = event.Display
	c.window = event.Window
	c.applyBackground()
	c.readSize()
	c.applyHints()
	c.Place()
}

func (c *Controller) SetBackground(background color.NRGBA) {
	c.bg = background
}

func (c *Controller) SetSize(size image.Point) {
	c.size = size
	c.Place()
}

func (c *Controller) Place() {
	if c.display == nil || c.window == 0 || c.size.X <= 0 || c.size.Y <= 0 || c.placed {
		return
	}
	display := (*C.Display)(c.display)
	window := C.Window(c.window)
	screen, work := workarea(display)
	pos := menuPosition(screen, work, c.size)
	if matePos, ok := matePanelPosition(screen, c.size); ok {
		pos = matePos
	}

	C.XMoveWindow(display, window, C.int(pos.X), C.int(pos.Y))
	C.XRaiseWindow(display, window)
	C.XFlush(display)
	c.placed = true
}

func (c *Controller) applyHints() {
	display := (*C.Display)(c.display)
	window := C.Window(c.window)
	c.applyNoDecorations(display, window)
	c.applyMenuType(display, window)
	c.applyTaskbarHints(display, window)
	C.XFlush(display)
}

func (c *Controller) applyNoDecorations(display *C.Display, window C.Window) {
	C.setNoDecorations(display, window)
}

func (c *Controller) applyMenuType(display *C.Display, window C.Window) {
	property := C.CString("_NET_WM_WINDOW_TYPE")
	defer C.free(unsafe.Pointer(property))
	atom := C.CString("_NET_WM_WINDOW_TYPE_MENU")
	defer C.free(unsafe.Pointer(atom))
	C.setAtomProperty(display, window, property, atom)
}

func (c *Controller) applyTaskbarHints(display *C.Display, window C.Window) {
	C.setMenuWindowState(display, window)
}

func (c *Controller) applyBackground() {
	if c.bg.A == 0 || c.display == nil || c.window == 0 {
		return
	}
	C.setWindowBackground(
		(*C.Display)(c.display),
		C.Window(c.window),
		C.uchar(c.bg.R),
		C.uchar(c.bg.G),
		C.uchar(c.bg.B),
	)
}

func (c *Controller) readSize() {
	if c.size.X > 0 && c.size.Y > 0 {
		return
	}
	var width, height C.int
	if C.getWindowSize((*C.Display)(c.display), C.Window(c.window), &width, &height) == 0 {
		return
	}
	c.size = image.Pt(int(width), int(height))
}

func workarea(display *C.Display) (image.Rectangle, image.Rectangle) {
	screenID := C.XDefaultScreen(display)
	root := C.XRootWindow(display, screenID)
	screen := image.Rect(0, 0, int(C.XDisplayWidth(display, screenID)), int(C.XDisplayHeight(display, screenID)))
	work := screen

	desktopValues := make([]C.ulong, 1)
	desktop := 0
	name := C.CString("_NET_CURRENT_DESKTOP")
	if C.getCardinalProperty(display, root, name, &desktopValues[0], 1) == 1 {
		desktop = int(desktopValues[0])
	}
	C.free(unsafe.Pointer(name))

	values := make([]C.ulong, 64)
	name = C.CString("_NET_WORKAREA")
	count := int(C.getCardinalProperty(display, root, name, &values[0], C.int(len(values))))
	C.free(unsafe.Pointer(name))
	if count >= 4 {
		offset := desktop * 4
		if offset+3 >= count {
			offset = 0
		}
		x := int(values[offset])
		y := int(values[offset+1])
		width := int(values[offset+2])
		height := int(values[offset+3])
		if width > 0 && height > 0 {
			work = image.Rect(x, y, x+width, y+height)
		}
	}
	return screen, work
}

func matePanelPosition(screen image.Rectangle, size image.Point) (image.Point, bool) {
	for _, id := range matePanelIDs() {
		p, ok := matePanel(id)
		if !ok {
			continue
		}
		if pos, ok := menuPositionForPanel(screen, p, size); ok {
			return pos, true
		}
	}
	return image.Point{}, false
}

func matePanelIDs() []string {
	value, ok := gsettings("get", "org.mate.panel", "toplevel-id-list")
	if !ok {
		return nil
	}
	value = strings.Trim(strings.TrimSpace(value), "[]")
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		id := strings.Trim(strings.TrimSpace(part), "'\"")
		if id != "" {
			result = append(result, id)
		}
	}
	return result
}

func matePanel(id string) (panel, bool) {
	schema := "org.mate.panel.toplevel:/org/mate/panel/toplevels/" + id + "/"
	orientation, ok := gsettings("get", schema, "orientation")
	if !ok {
		return panel{}, false
	}
	size, ok := gsettingsInt(schema, "size")
	if !ok {
		return panel{}, false
	}
	x, _ := gsettingsInt(schema, "x")
	y, _ := gsettingsInt(schema, "y")
	autoHide := false
	if value, ok := gsettings("get", schema, "auto-hide"); ok {
		autoHide = strings.TrimSpace(value) == "true"
	}
	return panel{
		orientation: strings.Trim(strings.TrimSpace(orientation), "'\""),
		autoHide:    autoHide,
		x:           x,
		y:           y,
		size:        size,
	}, true
}

func gsettingsInt(schema, key string) (int, bool) {
	value, ok := gsettings("get", schema, key)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	return n, err == nil
}

func gsettings(args ...string) (string, bool) {
	out, err := exec.Command("gsettings", args...).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}
