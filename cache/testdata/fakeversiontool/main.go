// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

// Command fakeversiontool is a native stand-in for a quality tool in the
// version cache tests. It logs every invocation to $GZQ_FAKE_VERSION_TOOL_LOG
// and answers --version with $GZQ_FAKE_VERSION_TOOL_VERSION.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	log, err := os.OpenFile(os.Getenv("GZQ_FAKE_VERSION_TOOL_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(3)
	}
	fmt.Fprintln(log, strings.Join(os.Args[1:], " "))
	_ = log.Close()
	if len(os.Args) < 2 || os.Args[1] != "--version" {
		os.Exit(2)
	}
	fmt.Println(os.Getenv("GZQ_FAKE_VERSION_TOOL_VERSION"))
}
