package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRestoreTimeoutStockScriptIsIdempotent(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	ctx := context.Background()

	require.NoError(t, client.Set(ctx, "{stock}:1", 5, 0).Err())
	for i := 0; i < 2; i++ {
		result, err := restoreTimeoutStockScript.Run(
			ctx,
			client,
			[]string{"{stock}:timeout-restored:order-1", "{stock}:1"},
			2,
			int64((30*24*time.Hour)/time.Second),
		).Int()
		require.NoError(t, err)
		require.Equal(t, 1-i, result)
	}

	stock, err := client.Get(ctx, "{stock}:1").Int()
	require.NoError(t, err)
	require.Equal(t, 7, stock)
}

func TestRestoreTimeoutStockScriptMarksMissingCacheWithoutCreatingIt(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	ctx := context.Background()

	result, err := restoreTimeoutStockScript.Run(
		ctx,
		client,
		[]string{"{stock}:timeout-restored:order-2", "{stock}:2"},
		3,
		int64((30*24*time.Hour)/time.Second),
	).Int()
	require.NoError(t, err)
	require.Equal(t, 1, result)
	require.Equal(t, int64(0), client.Exists(ctx, "{stock}:2").Val())
	require.Equal(t, int64(1), client.Exists(ctx, "{stock}:timeout-restored:order-2").Val())
}

func TestRestoreTimeoutStockScriptRejectsMalformedStock(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	ctx := context.Background()

	require.NoError(t, client.Set(ctx, "{stock}:3", "not-a-number", 0).Err())
	result, err := restoreTimeoutStockScript.Run(
		ctx,
		client,
		[]string{"{stock}:timeout-restored:order-3", "{stock}:3"},
		1,
		int64((30*24*time.Hour)/time.Second),
	).Int()
	require.NoError(t, err)
	require.Equal(t, -1, result)
	require.Equal(t, int64(0), client.Exists(ctx, "{stock}:timeout-restored:order-3").Val())
}
