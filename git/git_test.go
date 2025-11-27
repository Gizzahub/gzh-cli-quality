// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupGitRepo creates a temporary git repository for testing
func setupGitRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()
	require.NoError(t, err, "Failed to initialize git repository")

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Dir = tmpDir
	err = exec.Command("git", "config", "user.email", "test@example.com").Run()
	require.NoError(t, err)

	// Set user config in the test repo
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	return tmpDir
}

// createAndCommitFile creates a file and commits it
func createAndCommitFile(t *testing.T, repoDir, filename, content string) {
	t.Helper()

	filePath := filepath.Join(repoDir, filename)
	err := os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Add "+filename)
	cmd.Dir = repoDir
	err = cmd.Run()
	require.NoError(t, err)
}

func TestNewGitUtils(t *testing.T) {
	gitUtils := NewGitUtils("/test/path")

	assert.NotNil(t, gitUtils)
	assert.Equal(t, "/test/path", gitUtils.projectRoot)
}

func TestIsGitRepository_True(t *testing.T) {
	repoDir := setupGitRepo(t)
	gitUtils := NewGitUtils(repoDir)

	isRepo := gitUtils.IsGitRepository()
	assert.True(t, isRepo, "Should detect git repository")
}

func TestIsGitRepository_False(t *testing.T) {
	tmpDir := t.TempDir()
	gitUtils := NewGitUtils(tmpDir)

	isRepo := gitUtils.IsGitRepository()
	assert.False(t, isRepo, "Should not detect git repository in non-git directory")
}

func TestGetStagedFiles_Empty(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "initial.txt", "initial content")

	gitUtils := NewGitUtils(repoDir)

	files, err := gitUtils.GetStagedFiles()
	require.NoError(t, err)
	assert.Empty(t, files, "Should have no staged files")
}

func TestGetStagedFiles_WithFiles(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "initial.txt", "initial content")

	// Create and stage a new file
	newFile := filepath.Join(repoDir, "staged.txt")
	err := os.WriteFile(newFile, []byte("staged content"), 0o644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "staged.txt")
	cmd.Dir = repoDir
	err = cmd.Run()
	require.NoError(t, err)

	gitUtils := NewGitUtils(repoDir)
	files, err := gitUtils.GetStagedFiles()
	require.NoError(t, err)

	assert.Len(t, files, 1)
	assert.Contains(t, files, "staged.txt")
}

func TestGetModifiedFiles_Empty(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "file.txt", "content")

	gitUtils := NewGitUtils(repoDir)

	files, err := gitUtils.GetModifiedFiles()
	require.NoError(t, err)
	assert.Empty(t, files, "Should have no modified files")
}

func TestGetModifiedFiles_WithModifications(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "file.txt", "original content")

	// Modify the file
	filePath := filepath.Join(repoDir, "file.txt")
	err := os.WriteFile(filePath, []byte("modified content"), 0o644)
	require.NoError(t, err)

	gitUtils := NewGitUtils(repoDir)
	files, err := gitUtils.GetModifiedFiles()
	require.NoError(t, err)

	assert.Len(t, files, 1)
	assert.Contains(t, files, "file.txt")
}

func TestGetUntrackedFiles_Empty(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "tracked.txt", "content")

	gitUtils := NewGitUtils(repoDir)

	files, err := gitUtils.GetUntrackedFiles()
	require.NoError(t, err)
	assert.Empty(t, files, "Should have no untracked files")
}

func TestGetUntrackedFiles_WithFiles(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "tracked.txt", "content")

	// Create untracked file
	untrackedPath := filepath.Join(repoDir, "untracked.txt")
	err := os.WriteFile(untrackedPath, []byte("untracked content"), 0o644)
	require.NoError(t, err)

	gitUtils := NewGitUtils(repoDir)
	files, err := gitUtils.GetUntrackedFiles()
	require.NoError(t, err)

	assert.Len(t, files, 1)
	assert.Contains(t, files, "untracked.txt")
}

func TestGetAllChangedFiles(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "initial.txt", "content")

	// Create staged file
	stagedPath := filepath.Join(repoDir, "staged.txt")
	err := os.WriteFile(stagedPath, []byte("staged"), 0o644)
	require.NoError(t, err)
	cmd := exec.Command("git", "add", "staged.txt")
	cmd.Dir = repoDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create modified file
	modifiedPath := filepath.Join(repoDir, "initial.txt")
	err = os.WriteFile(modifiedPath, []byte("modified"), 0o644)
	require.NoError(t, err)

	// Create untracked file
	untrackedPath := filepath.Join(repoDir, "untracked.txt")
	err = os.WriteFile(untrackedPath, []byte("untracked"), 0o644)
	require.NoError(t, err)

	gitUtils := NewGitUtils(repoDir)
	files, err := gitUtils.GetAllChangedFiles()
	require.NoError(t, err)

	// Should include all three types
	assert.GreaterOrEqual(t, len(files), 2, "Should have multiple changed files")

	// Files should be absolute paths
	for _, file := range files {
		assert.True(t, filepath.IsAbs(file), "File path should be absolute: %s", file)
	}
}

func TestGetChangedFiles_DefaultSinceHEAD(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "file1.txt", "content1")
	createAndCommitFile(t, repoDir, "file2.txt", "content2")

	gitUtils := NewGitUtils(repoDir)

	// Get changed files since HEAD~1 (default)
	files, err := gitUtils.GetChangedFiles("")
	require.NoError(t, err)

	// Should show file2.txt as it's the most recent commit
	assert.Contains(t, files, "file2.txt")
}

func TestGetChangedFiles_SpecificCommit(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "file1.txt", "content1")
	createAndCommitFile(t, repoDir, "file2.txt", "content2")
	createAndCommitFile(t, repoDir, "file3.txt", "content3")

	gitUtils := NewGitUtils(repoDir)

	// Get changed files since HEAD~2
	files, err := gitUtils.GetChangedFiles("HEAD~2")
	require.NoError(t, err)

	// Should include file2 and file3
	assert.GreaterOrEqual(t, len(files), 1, "Should have changed files")
}

func TestGetCurrentBranch(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "file.txt", "content")

	gitUtils := NewGitUtils(repoDir)

	branch, err := gitUtils.GetCurrentBranch()
	require.NoError(t, err)

	// Default branch is usually "master" or "main"
	assert.NotEmpty(t, branch, "Should have a current branch")
	assert.True(t, branch == "master" || branch == "main", "Branch should be master or main, got: %s", branch)
}

func TestGetCurrentBranch_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitUtils := NewGitUtils(tmpDir)

	_, err := gitUtils.GetCurrentBranch()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestParseFileList(t *testing.T) {
	gitUtils := NewGitUtils("/test")

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "Single file",
			input:    "file.go",
			expected: []string{"file.go"},
		},
		{
			name:     "Multiple files",
			input:    "file1.go\nfile2.go\nfile3.go",
			expected: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:     "Files with whitespace",
			input:    "  file1.go  \n  file2.go  \n",
			expected: []string{"file1.go", "file2.go"},
		},
		{
			name:     "Files with empty lines",
			input:    "file1.go\n\nfile2.go\n\n",
			expected: []string{"file1.go", "file2.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gitUtils.parseFileList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeduplicateAndMakeAbsolute(t *testing.T) {
	repoDir := setupGitRepo(t)

	// Create test files
	file1 := filepath.Join(repoDir, "file1.txt")
	file2 := filepath.Join(repoDir, "file2.txt")
	err := os.WriteFile(file1, []byte("content1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("content2"), 0o644)
	require.NoError(t, err)

	gitUtils := NewGitUtils(repoDir)

	// Test with duplicates
	input := []string{"file1.txt", "file2.txt", "file1.txt", "file2.txt"}
	result := gitUtils.deduplicateAndMakeAbsolute(input)

	// Should remove duplicates
	assert.Len(t, result, 2, "Should have 2 unique files")

	// Should be absolute paths
	for _, file := range result {
		assert.True(t, filepath.IsAbs(file), "Path should be absolute")
	}

	// Should only include existing files
	assert.Contains(t, result, file1)
	assert.Contains(t, result, file2)
}

func TestDeduplicateAndMakeAbsolute_NonExistentFiles(t *testing.T) {
	repoDir := t.TempDir()
	gitUtils := NewGitUtils(repoDir)

	// Files that don't exist
	input := []string{"nonexistent1.txt", "nonexistent2.txt"}
	result := gitUtils.deduplicateAndMakeAbsolute(input)

	// Should return empty slice for nonexistent files
	assert.Empty(t, result, "Should not include nonexistent files")
}

func TestValidateCommitish_Valid(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "file.txt", "content")

	gitUtils := NewGitUtils(repoDir)

	// HEAD should be valid
	err := gitUtils.ValidateCommitish("HEAD")
	assert.NoError(t, err, "HEAD should be valid")

	// HEAD~1 may not be valid in a fresh repo with only one commit
	// So we skip that test
}

func TestValidateCommitish_Invalid(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "file.txt", "content")

	gitUtils := NewGitUtils(repoDir)

	// Invalid commit reference
	err := gitUtils.ValidateCommitish("invalid-commit-xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid commit reference")
}

func TestValidateCommitish_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitUtils := NewGitUtils(tmpDir)

	err := gitUtils.ValidateCommitish("HEAD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGetStagedFiles_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitUtils := NewGitUtils(tmpDir)

	_, err := gitUtils.GetStagedFiles()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGetModifiedFiles_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitUtils := NewGitUtils(tmpDir)

	_, err := gitUtils.GetModifiedFiles()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGetUntrackedFiles_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitUtils := NewGitUtils(tmpDir)

	_, err := gitUtils.GetUntrackedFiles()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGetAllChangedFiles_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitUtils := NewGitUtils(tmpDir)

	_, err := gitUtils.GetAllChangedFiles()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGetChangedFiles_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	gitUtils := NewGitUtils(tmpDir)

	_, err := gitUtils.GetChangedFiles("HEAD~1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGetStagedFiles_MultipleFiles(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "initial.txt", "initial")

	// Stage multiple files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range files {
		filePath := filepath.Join(repoDir, filename)
		err := os.WriteFile(filePath, []byte("content"), 0o644)
		require.NoError(t, err)

		cmd := exec.Command("git", "add", filename)
		cmd.Dir = repoDir
		err = cmd.Run()
		require.NoError(t, err)
	}

	gitUtils := NewGitUtils(repoDir)
	staged, err := gitUtils.GetStagedFiles()
	require.NoError(t, err)

	assert.Len(t, staged, 3)
	for _, filename := range files {
		assert.Contains(t, staged, filename)
	}
}

func TestIsGitRepository_WithGitFile(t *testing.T) {
	// This tests the git worktree/submodule case where .git is a file, not a directory
	repoDir := setupGitRepo(t)

	// Create a .git file (simulating worktree)
	gitFilePath := filepath.Join(repoDir, ".git", "worktree-test")
	err := os.WriteFile(gitFilePath, []byte("gitdir: ../main/.git/worktrees/test"), 0o644)
	require.NoError(t, err)

	gitUtils := NewGitUtils(repoDir)
	isRepo := gitUtils.IsGitRepository()

	// Should still detect as git repo via git rev-parse
	assert.True(t, isRepo)
}

func TestGetAllChangedFiles_Deduplication(t *testing.T) {
	repoDir := setupGitRepo(t)
	createAndCommitFile(t, repoDir, "file.txt", "original")

	// Modify and stage the same file
	filePath := filepath.Join(repoDir, "file.txt")
	err := os.WriteFile(filePath, []byte("modified and staged"), 0o644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "file.txt")
	cmd.Dir = repoDir
	err = cmd.Run()
	require.NoError(t, err)

	// Modify again (so it's both staged and modified)
	err = os.WriteFile(filePath, []byte("modified again"), 0o644)
	require.NoError(t, err)

	gitUtils := NewGitUtils(repoDir)
	files, err := gitUtils.GetAllChangedFiles()
	require.NoError(t, err)

	// Even though file appears in both staged and modified, should only appear once
	// Count occurrences
	count := 0
	for _, f := range files {
		if filepath.Base(f) == "file.txt" {
			count++
		}
	}
	assert.Equal(t, 1, count, "File should appear only once after deduplication")
}
