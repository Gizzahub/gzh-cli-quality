// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// GenerateKey generates a cache key for a file and tool combination.
func GenerateKey(filePath string, tool tools.QualityTool, options tools.ExecuteOptions) (CacheKey, error) {
	// 1. Calculate file hash
	fileHash, err := hashFile(filePath)
	if err != nil {
		return CacheKey{}, fmt.Errorf("failed to hash file %s: %w", filePath, err)
	}

	// 2. Get tool version
	toolVersion, err := tool.GetVersion()
	if err != nil {
		// If version cannot be determined, use "unknown"
		// This will cause cache misses, which is safe
		toolVersion = "unknown"
	}

	// 3. Calculate config hash
	configFiles := tool.FindConfigFiles(options.ProjectRoot)
	configHash, err := hashFiles(configFiles)
	if err != nil {
		return CacheKey{}, fmt.Errorf("failed to hash config files: %w", err)
	}

	// 4. Calculate options hash
	optionsHash := hashOptions(options)

	// 5. Get absolute file path
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		absFilePath = filePath
	}

	return CacheKey{
		FilePath:    absFilePath,
		FileHash:    fileHash,
		ToolName:    tool.Name(),
		ToolVersion: toolVersion,
		ConfigHash:  configHash,
		OptionsHash: optionsHash,
	}, nil
}

// hashFile calculates SHA256 hash of a file's content.
func hashFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// hashFiles calculates SHA256 hash of multiple files combined.
// Returns empty string if no files or all files don't exist.
func hashFiles(filePaths []string) (string, error) {
	if len(filePaths) == 0 {
		return "", nil
	}

	// Sort file paths for deterministic hashing
	sortedPaths := make([]string, len(filePaths))
	copy(sortedPaths, filePaths)
	sort.Strings(sortedPaths)

	hasher := sha256.New()

	// Hash each file's content
	for _, path := range sortedPaths {
		content, err := os.ReadFile(path)
		if err != nil {
			// If config file doesn't exist, skip it
			// This is common (e.g., .prettierrc may not exist)
			continue
		}

		// Write file path and content to hasher
		hasher.Write([]byte(path))
		hasher.Write(content)
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash), nil
}

// hashOptions calculates SHA256 hash of execution options.
func hashOptions(options tools.ExecuteOptions) string {
	hasher := sha256.New()

	// Hash relevant fields in deterministic order
	// ProjectRoot intentionally excluded (path-specific)
	// ConfigFile excluded (already in config hash)

	// Fix flag
	if options.Fix {
		hasher.Write([]byte("fix:true"))
	}

	// FormatOnly flag
	if options.FormatOnly {
		hasher.Write([]byte("format-only:true"))
	}

	// LintOnly flag
	if options.LintOnly {
		hasher.Write([]byte("lint-only:true"))
	}

	// ExtraArgs (sorted for determinism)
	if len(options.ExtraArgs) > 0 {
		sortedArgs := make([]string, len(options.ExtraArgs))
		copy(sortedArgs, options.ExtraArgs)
		sort.Strings(sortedArgs)

		hasher.Write([]byte("args:"))
		hasher.Write([]byte(strings.Join(sortedArgs, ",")))
	}

	// Env variables (sorted by key)
	if len(options.Env) > 0 {
		var keys []string
		for k := range options.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Fprintf(hasher, "env:%s=%s", k, options.Env[k])
		}
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}

// ValidateKey validates that a cache key is well-formed.
func ValidateKey(key CacheKey) error {
	if key.FilePath == "" {
		return fmt.Errorf("file path is empty")
	}

	if key.FileHash == "" {
		return fmt.Errorf("file hash is empty")
	}

	if key.ToolName == "" {
		return fmt.Errorf("tool name is empty")
	}

	if key.ToolVersion == "" {
		return fmt.Errorf("tool version is empty")
	}

	// ConfigHash and OptionsHash can be empty (no config/options)

	return nil
}
