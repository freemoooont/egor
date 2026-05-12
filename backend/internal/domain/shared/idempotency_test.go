package shared_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/domain/shared"
)

func TestIdempotencyKey_IsZeroAndString(t *testing.T) {
	require.True(t, shared.IdempotencyKey("").IsZero())
	require.True(t, shared.IdempotencyKey("   ").IsZero())
	require.False(t, shared.IdempotencyKey("k-123").IsZero())
	require.Equal(t, "k-123", shared.IdempotencyKey("k-123").String())
}
