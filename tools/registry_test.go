// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock tool for testing
type mockTool struct {
	name     string
	language string
	toolType ToolType
}

func (m *mockTool) Name() string       { return m.name }
func (m *mockTool) Language() string   { return m.language }
func (m *mockTool) Type() ToolType     { return m.toolType }
func (m *mockTool) IsAvailable() bool  { return true }
func (m *mockTool) Install() error     { return nil }
func (m *mockTool) GetVersion() (string, error) { return "1.0.0", nil }
func (m *mockTool) Upgrade() error     { return nil }
func (m *mockTool) FindConfigFiles(projectRoot string) []string { return nil }
func (m *mockTool) Execute(ctx context.Context, files []string, options ExecuteOptions) (*Result, error) {
	return &Result{Tool: m.name, Success: true}, nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.tools)
	assert.Equal(t, 0, len(registry.tools))
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	tool := &mockTool{name: "gofmt", language: "Go", toolType: FORMAT}

	registry.Register(tool)

	assert.Equal(t, 1, len(registry.tools))
	assert.Equal(t, tool, registry.tools["gofmt"])
}

func TestRegistry_RegisterMultiple(t *testing.T) {
	registry := NewRegistry()

	tool1 := &mockTool{name: "gofmt", language: "Go", toolType: FORMAT}
	tool2 := &mockTool{name: "golint", language: "Go", toolType: LINT}
	tool3 := &mockTool{name: "black", language: "Python", toolType: FORMAT}

	registry.Register(tool1)
	registry.Register(tool2)
	registry.Register(tool3)

	assert.Equal(t, 3, len(registry.tools))
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRegistry()

	tool1 := &mockTool{name: "gofmt", language: "Go", toolType: FORMAT}
	tool2 := &mockTool{name: "gofmt", language: "Go", toolType: BOTH} // Different type, same name

	registry.Register(tool1)
	registry.Register(tool2)

	// Should overwrite the first registration
	assert.Equal(t, 1, len(registry.tools))
	assert.Equal(t, tool2, registry.tools["gofmt"])
	assert.Equal(t, BOTH, registry.tools["gofmt"].Type())
}

func TestRegistry_GetTools(t *testing.T) {
	registry := NewRegistry()

	tool1 := &mockTool{name: "gofmt", language: "Go", toolType: FORMAT}
	tool2 := &mockTool{name: "golint", language: "Go", toolType: LINT}

	registry.Register(tool1)
	registry.Register(tool2)

	tools := registry.GetTools()

	assert.Equal(t, 2, len(tools))

	// Check both tools are present (order may vary)
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}
	assert.True(t, toolNames["gofmt"])
	assert.True(t, toolNames["golint"])
}

func TestRegistry_GetTools_Empty(t *testing.T) {
	registry := NewRegistry()

	tools := registry.GetTools()

	assert.NotNil(t, tools)
	assert.Equal(t, 0, len(tools))
}

func TestRegistry_GetToolsByLanguage(t *testing.T) {
	registry := NewRegistry()

	goTool1 := &mockTool{name: "gofmt", language: "Go", toolType: FORMAT}
	goTool2 := &mockTool{name: "golint", language: "Go", toolType: LINT}
	pyTool := &mockTool{name: "black", language: "Python", toolType: FORMAT}
	jsTool := &mockTool{name: "eslint", language: "JavaScript", toolType: LINT}

	registry.Register(goTool1)
	registry.Register(goTool2)
	registry.Register(pyTool)
	registry.Register(jsTool)

	t.Run("Go tools", func(t *testing.T) {
		tools := registry.GetToolsByLanguage("Go")
		assert.Equal(t, 2, len(tools))

		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name()] = true
			assert.Equal(t, "Go", tool.Language())
		}
		assert.True(t, toolNames["gofmt"])
		assert.True(t, toolNames["golint"])
	})

	t.Run("Python tools", func(t *testing.T) {
		tools := registry.GetToolsByLanguage("Python")
		assert.Equal(t, 1, len(tools))
		assert.Equal(t, "black", tools[0].Name())
	})

	t.Run("Non-existent language", func(t *testing.T) {
		tools := registry.GetToolsByLanguage("Rust")
		assert.Equal(t, 0, len(tools))
	})
}

func TestRegistry_GetToolsByType(t *testing.T) {
	registry := NewRegistry()

	formatTool := &mockTool{name: "gofmt", language: "Go", toolType: FORMAT}
	lintTool := &mockTool{name: "golint", language: "Go", toolType: LINT}
	bothTool := &mockTool{name: "golangci-lint", language: "Go", toolType: BOTH}

	registry.Register(formatTool)
	registry.Register(lintTool)
	registry.Register(bothTool)

	t.Run("FORMAT type", func(t *testing.T) {
		tools := registry.GetToolsByType(FORMAT)
		// Should return formatTool and bothTool
		assert.Equal(t, 2, len(tools))

		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name()] = true
		}
		assert.True(t, toolNames["gofmt"])
		assert.True(t, toolNames["golangci-lint"])
	})

	t.Run("LINT type", func(t *testing.T) {
		tools := registry.GetToolsByType(LINT)
		// Should return lintTool and bothTool
		assert.Equal(t, 2, len(tools))

		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name()] = true
		}
		assert.True(t, toolNames["golint"])
		assert.True(t, toolNames["golangci-lint"])
	})

	t.Run("BOTH type", func(t *testing.T) {
		tools := registry.GetToolsByType(BOTH)
		// Should return only bothTool
		assert.Equal(t, 1, len(tools))
		assert.Equal(t, "golangci-lint", tools[0].Name())
	})
}

func TestRegistry_FindTool(t *testing.T) {
	registry := NewRegistry()

	tool := &mockTool{name: "gofmt", language: "Go", toolType: FORMAT}
	registry.Register(tool)

	t.Run("Existing tool", func(t *testing.T) {
		found := registry.FindTool("gofmt")
		require.NotNil(t, found)
		assert.Equal(t, "gofmt", found.Name())
		assert.Equal(t, "Go", found.Language())
	})

	t.Run("Non-existent tool", func(t *testing.T) {
		found := registry.FindTool("nonexistent")
		assert.Nil(t, found)
	})
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	// Test concurrent registration and reads
	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(index int) {
			tool := &mockTool{
				name:     string(rune('A' + index)),
				language: "Go",
				toolType: FORMAT,
			}
			registry.Register(tool)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			_ = registry.GetTools()
			_ = registry.GetToolsByLanguage("Go")
			_ = registry.GetToolsByType(FORMAT)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify registry state
	tools := registry.GetTools()
	assert.Equal(t, 5, len(tools))
}

func TestToolType_String(t *testing.T) {
	tests := []struct {
		toolType ToolType
		expected string
	}{
		{FORMAT, "formatter"},
		{LINT, "linter"},
		{BOTH, "formatter+linter"},
		{ToolType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.toolType.String())
		})
	}
}
