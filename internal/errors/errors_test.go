package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorContextAndIdentity(t *testing.T) {
	cause := fmt.Errorf("runner: %w", ErrExecutionFailed)
	plain := New("lint", "ruff", cause)
	require.Equal(t, "lint: ruff failed: runner: execution failed", plain.Error())
	require.Same(t, cause, plain.Unwrap())
	wrapped := WrapWithPath("lint", "ruff", "a.py", cause)
	require.Equal(t, "lint: ruff failed for a.py: runner: execution failed", wrapped.Error())
	require.True(t, Is(wrapped, ErrExecutionFailed))
	require.False(t, Is(wrapped, ErrInvalidConfig))
	var detail *QualityError
	require.True(t, As(wrapped, &detail))
	require.Equal(t, "a.py", detail.Path)
	require.True(t, Is(Wrap("lint", "ruff", cause), ErrExecutionFailed))
	require.Nil(t, Wrap("lint", "ruff", nil))
	require.Nil(t, WrapWithPath("lint", "ruff", "a.py", nil))
}
