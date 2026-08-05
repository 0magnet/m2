//go:build !js
package main

// Origin returns "scheme://host[:port]" (e.g., "https://example.net").
func Origin() string { return "" }
// Host returns "domain[:port]" (e.g., "example.net" or "127.0.0.1:8080").
func Host() string { return "" }
