//go:build !linux || android

package windowctrl

import (
	"image"
	"image/color"
)

type Controller struct{}

func (c *Controller) HandleEvent(any)           {}
func (c *Controller) SetBackground(color.NRGBA) {}
func (c *Controller) SetSize(image.Point)       {}
func (c *Controller) Place()                    {}
