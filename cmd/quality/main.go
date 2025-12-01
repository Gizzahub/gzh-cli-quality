// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	"github.com/Gizzahub/gzh-cli-quality"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := quality.NewQualityCmd()
	rootCmd.Use = "gz-quality"
	rootCmd.Short = "Multi-language code quality tool orchestrator"
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
