// Package main ☠.go
package main

import (
	"embed"
)

//go:embed content/*
var econtent embed.FS

//go:embed products.json
var productsJSON string

//go:embed core.toml
var coreToml string
