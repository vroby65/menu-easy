//go:build !linux || android

package windowctrl

import "image"

type Controller struct{}

func (c *Controller) HandleEvent(any)     {}
func (c *Controller) SetSize(image.Point) {}
func (c *Controller) Place()              {}
