package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisCache_SetGetAndMissing(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	c, err := NewCache(mr.Host(), mr.Port())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Close()) })

	ctx := context.Background()
	require.NoError(t, c.SetKeyValWithTTL(ctx, "k1", "v1", time.Minute))

	v, err := c.GetKeyVal(ctx, "k1")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	missing, err := c.GetKeyVal(ctx, "missing")
	require.NoError(t, err)
	assert.Equal(t, "", missing)
}
