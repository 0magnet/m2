//go:build !js

package main

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

// StartCheckout (native): just inform that checkout is web-only for now.
func StartCheckout(_ *Cart, anchor core.Widget) {
	d := core.NewBody("Checkout")
	core.NewText(d).
		SetType(core.TextSupporting).
		SetText("Checkout is currently supported on the web build. Please open this site in your browser to complete payment.")

	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar).SetIcon(icons.Close).OnClick(func(e events.Event) {
			// no-op
		})
		d.AddOK(bar).SetIcon(icons.Home).SetText("OK")
	})
	d.RunDialog(anchor)
}
