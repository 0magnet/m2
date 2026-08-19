//go:build js

package main

import (
	"syscall/js"
)

// currentLocation tries window/self and returns the JS location object.
func currentLocation() js.Value {
	g := js.Global()
	if l := g.Get("location"); l.Truthy() {
		return l
	}
	if w := g.Get("window"); w.Truthy() {
		if l := w.Get("location"); l.Truthy() {
			return l
		}
	}
	if s := g.Get("self"); s.Truthy() {
		if l := s.Get("location"); l.Truthy() {
			return l
		}
	}
	return js.Undefined()
}

// Origin returns "scheme://host[:port]" (e.g., "https://example.net").
func Origin() string {
	loc := currentLocation()
	if !loc.Truthy() {
		return ""
	}
	if o := loc.Get("origin").String(); o != "" {
		return o
	}
	return loc.Get("protocol").String() + "//" + Host()
}

// Host returns "domain[:port]" (e.g., "example.net" or "127.0.0.1:8080").
func Host() string {
	loc := currentLocation()
	if !loc.Truthy() {
		return ""
	}
	// location.host already includes port if present
	h := loc.Get("host").String()
	if h != "" {
		return h
	}
	// Fallback if host is empty (file: URLs etc.)
	hn := loc.Get("hostname").String()
	pt := loc.Get("port").String()
	if hn == "" {
		return ""
	}
	if pt != "" {
		return hn + ":" + pt
	}
	return hn
}
