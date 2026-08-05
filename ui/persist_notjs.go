//go:build !js

package main

// No-op persistence for non-web builds.
func LoadCartState() (map[string]*CartItem, bool) { return nil, false }
func SaveCartState(_ *Cart)                        {}
