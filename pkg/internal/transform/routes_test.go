package transform

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/ebpf-autoinstrument/pkg/internal/testutil"
)

const testTimeout = 5 * time.Second

func TestUnmatchedWildcard(t *testing.T) {
	for _, tc := range []UnmatchType{"", UnmatchWildcard, "invalid_value"} {
		t.Run(string(tc), func(t *testing.T) {
			router, err := RoutesProvider(context.TODO(), &RoutesConfig{Unmatch: tc, Patterns: []string{"/user/:id"}})
			require.NoError(t, err)
			in, out := make(chan []HTTPRequestSpan, 10), make(chan []HTTPRequestSpan, 10)
			defer close(in)
			go router(in, out)
			in <- []HTTPRequestSpan{{Path: "/user/1234"}}
			assert.Equal(t, []HTTPRequestSpan{{
				Path:  "/user/1234",
				Route: "/user/:id",
			}}, testutil.ReadChannel(t, out, testTimeout))
			in <- []HTTPRequestSpan{{Path: "/some/path"}}
			assert.Equal(t, []HTTPRequestSpan{{
				Path:  "/some/path",
				Route: "*",
			}}, testutil.ReadChannel(t, out, testTimeout))
		})
	}
}

func TestUnmatchedPath(t *testing.T) {
	router, err := RoutesProvider(context.TODO(), &RoutesConfig{Unmatch: UnmatchPath, Patterns: []string{"/user/:id"}})
	require.NoError(t, err)
	in, out := make(chan []HTTPRequestSpan, 10), make(chan []HTTPRequestSpan, 10)
	defer close(in)
	go router(in, out)
	in <- []HTTPRequestSpan{{Path: "/user/1234"}}
	assert.Equal(t, []HTTPRequestSpan{{
		Path:  "/user/1234",
		Route: "/user/:id",
	}}, testutil.ReadChannel(t, out, testTimeout))
	in <- []HTTPRequestSpan{{Path: "/some/path"}}
	assert.Equal(t, []HTTPRequestSpan{{
		Path:  "/some/path",
		Route: "/some/path",
	}}, testutil.ReadChannel(t, out, testTimeout))
}

func TestUnmatchedEmpty(t *testing.T) {
	router, err := RoutesProvider(context.TODO(), &RoutesConfig{Unmatch: UnmatchUnset, Patterns: []string{"/user/:id"}})
	require.NoError(t, err)
	in, out := make(chan []HTTPRequestSpan, 10), make(chan []HTTPRequestSpan, 10)
	defer close(in)
	go router(in, out)
	in <- []HTTPRequestSpan{{Path: "/user/1234"}}
	assert.Equal(t, []HTTPRequestSpan{{
		Path:  "/user/1234",
		Route: "/user/:id",
	}}, testutil.ReadChannel(t, out, testTimeout))
	in <- []HTTPRequestSpan{{Path: "/some/path"}}
	assert.Equal(t, []HTTPRequestSpan{{
		Path: "/some/path",
	}}, testutil.ReadChannel(t, out, testTimeout))
}
