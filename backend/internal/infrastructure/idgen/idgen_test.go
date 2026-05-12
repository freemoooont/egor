package idgen_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/micocards/api/internal/infrastructure/idgen"
)

func TestUUID_GeneratesNonEmptyDistinctIDs(t *testing.T) {
	a := idgen.UUID{}.NewID(context.Background())
	b := idgen.UUID{}.NewID(context.Background())
	require.NotEmpty(t, a)
	require.NotEmpty(t, b)
	require.NotEqual(t, a, b)
}

func TestSequential_IncrementsWithPrefix(t *testing.T) {
	g := idgen.NewSequential("u")
	require.Equal(t, "u-1", g.NewID(context.Background()))
	require.Equal(t, "u-2", g.NewID(context.Background()))
}

func TestStatic_ReturnsSuppliedIDs(t *testing.T) {
	g := idgen.NewStatic("a", "b", "c")
	require.Equal(t, "a", g.NewID(context.Background()))
	require.Equal(t, "b", g.NewID(context.Background()))
	require.Equal(t, "c", g.NewID(context.Background()))
	// once drained, returns the last id
	require.Equal(t, "c", g.NewID(context.Background()))

	empty := idgen.NewStatic()
	require.Equal(t, "", empty.NewID(context.Background()))
}
