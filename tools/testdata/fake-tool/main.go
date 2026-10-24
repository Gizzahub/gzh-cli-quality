// Command fake-tool is a native executable fixture for adapter subprocess tests.
// It is built explicitly by adapters_test.go on each tested operating system.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	executable, err := os.Executable()
	if err != nil {
		fail(err)
	}
	mode, err := os.ReadFile(executable + ".mode")
	if err != nil {
		fail(err)
	}
	switch string(mode) {
	case "observe":
		cwd, err := os.Getwd()
		if err != nil {
			fail(err)
		}
		var log strings.Builder
		fmt.Fprintf(&log, "cwd=%s\nenv=%s\n", cwd, os.Getenv("QUALITY_TEST_VALUE"))
		for _, arg := range os.Args[1:] {
			fmt.Fprintf(&log, "arg=%s\n", arg)
		}
		if err := os.WriteFile(os.Getenv("QUALITY_TEST_LOG"), []byte(log.String()), 0o600); err != nil {
			fail(err)
		}
	case "format":
		name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
		write := name == "black" || name == "rustfmt" || name == "cargo" || name == "ruff" || name == "taplo"
		for _, arg := range os.Args[1:] {
			switch arg {
			case "--check", "--dry-run", "--diff", "--exit-code":
				write = false
			case "-w", "--write", "-F", "write", "fix":
				write = true
			case "-i":
				if name != "shfmt" {
					write = true
				}
			}
		}
		if write {
			if err := os.WriteFile(os.Getenv("QUALITY_TEST_FILE"), []byte("formatted\n"), 0o600); err != nil {
				fail(err)
			}
		}
	case "diagnostic":
		fmt.Println("a.yaml:2:3: [error] bad value (syntax)")
		os.Exit(1)
	case "sleep":
		time.Sleep(10 * time.Second)
	case "difference":
		fmt.Println("input.go")
	case "version":
		fmt.Println(strings.Join(os.Args[1:], " "))
	case "fail":
		os.Exit(1)
	case "clean":
	default:
		fail(fmt.Errorf("unknown fixture mode %q", mode))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
