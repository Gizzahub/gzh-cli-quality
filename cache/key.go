// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// GenerateKey generates a cache key for a file and tool combination.
func GenerateKey(filePath string, tool tools.QualityTool, options tools.ExecuteOptions) (CacheKey, error) {
	// JSON storage normalizes invalid UTF-8. Bypass caching rather than merge
	// distinct argv/environment bytes or change a tool's actual inputs.
	values := make([]string, 0, 2+len(options.ExtraArgs)+2*len(options.Env))
	values = append(values, options.ProjectRoot, options.ConfigFile)
	values = append(values, options.ExtraArgs...)
	for key, value := range options.Env {
		values = append(values, key, value)
	}
	for _, value := range values {
		if !utf8.ValidString(value) {
			return CacheKey{}, fmt.Errorf("cache options contain invalid UTF-8")
		}
	}
	if !filepath.IsAbs(filePath) && options.ProjectRoot != "" {
		filePath = filepath.Join(options.ProjectRoot, filePath)
	}
	// 1. Calculate file hash
	fileHash, err := hashFile(filePath)
	if err != nil {
		return CacheKey{}, fmt.Errorf("failed to hash file %s: %w", filePath, err)
	}

	// 2. Get tool version
	toolVersion, err := tool.GetVersion()
	if err != nil {
		return CacheKey{}, fmt.Errorf("failed to determine tool version: %w", err)
	}
	if toolVersion == "" || toolVersion == "unknown" {
		return CacheKey{}, fmt.Errorf("tool version is unknown")
	}

	// 3. Calculate config hash
	configFiles := tool.FindConfigFiles(options.ProjectRoot)
	if options.ConfigFile != "" {
		configPath := options.ConfigFile
		if !filepath.IsAbs(configPath) {
			configPath = filepath.Join(options.ProjectRoot, configPath)
		}
		configFiles = append(configFiles, configPath)
	}
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

	key := CacheKey{
		FilePath:    absFilePath,
		FileHash:    fileHash,
		ToolName:    tool.Name(),
		ToolVersion: toolVersion,
		ConfigHash:  configHash,
		OptionsHash: optionsHash,
	}
	if err := ValidateKey(key); err != nil {
		return CacheKey{}, err
	}
	return key, nil
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
// Returns an error if an identified config cannot be read, so callers bypass cache.
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
			return "", fmt.Errorf("read config %s: %w", path, err)
		}

		// Write file path and content to hasher
		fmt.Fprintf(hasher, "%d:%s%d:%s", len(path), path, len(content), content)
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash), nil
}

// hashOptions calculates SHA256 hash of execution options.
func hashOptions(options tools.ExecuteOptions) string {
	// JSON frames each value, sorts environment map keys and preserves argv
	// order. Working directory and explicit config choice affect tool behavior.
	data, _ := json.Marshal(options) // ExecuteOptions contains only JSON-safe fields.
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ValidateKey validates that a cache key is well-formed.
func ValidateKey(key CacheKey) error {
	for _, value := range []string{key.FilePath, key.FileHash, key.ToolName, key.ToolVersion, key.ConfigHash, key.OptionsHash} {
		if !utf8.ValidString(value) {
			return fmt.Errorf("cache identity contains invalid UTF-8")
		}
	}
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
