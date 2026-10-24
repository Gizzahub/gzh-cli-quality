// Command fake-tool is a native executable fixture for adapter subprocess tests.
// It is built explicitly by adapters_test.go on each tested operating system.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	executable, err := os.Executable()
	if err != nil {
		fail(err)
	}
	if len(os.Args) == 2 && os.Args[1] == "--hold-version-pipe" {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(executable + ".release"); err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if err := os.WriteFile(executable+".child.done", []byte("done"), 0o600); err != nil {
			fail(err)
		}
		return
	}
	mode, err := os.ReadFile(executable + ".mode")
	if err != nil {
		fail(err)
	}
	switch string(mode) {
	case "hold-version-pipe":
		child := exec.Command(executable, "--hold-version-pipe")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			fail(err)
		}
		if err := os.WriteFile(executable+".child.pid", []byte(fmt.Sprint(child.Process.Pid)), 0o600); err != nil {
			fail(err)
		}
		time.Sleep(30 * time.Second)
	case "component-version":
		log, err := os.OpenFile(executable+".calls", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			fail(err)
		}
		_, writeErr := fmt.Fprintln(log, strings.Join(os.Args[1:], " "))
		closeErr := log.Close()
		if writeErr != nil {
			fail(writeErr)
		}
		if closeErr != nil {
			fail(closeErr)
		}
		standalone := len(os.Args) >= 2 && os.Args[1] == "--version"
		if standalone || (len(os.Args) >= 3 && os.Args[2] == "--version") {
			data, err := os.ReadFile(executable + ".versions")
			if err != nil {
				fail(err)
			}
			var versions map[string]string
			if local, err := os.ReadFile(".quality-component-versions"); err == nil {
				data = local
			}
			if err := json.Unmarshal(data, &versions); err != nil {
				fail(err)
			}
			key := os.Args[1]
			if standalone {
				key = "fmt"
			}
			if selected := os.Getenv("RUSTUP_TOOLCHAIN"); selected != "" {
				key += "@" + selected
			}
			fmt.Println(versions[key])
		}
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
		fmt.Fprintln(os.Stderr, "fake-tool: component not installed")
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
