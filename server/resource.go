//go:build !release

package main

import "embed"

// 静的リソース
//
//go:embed static/*
var static embed.FS
