package fallback

import (
	"testing"

	"github.com/code-koan/llm-sdk-go/internal/testutil"
	"github.com/code-koan/llm-sdk-go/providers"
	"github.com/stretchr/testify/require"
)

func TestRandomSelector_Select(t *testing.T) {
	t.Parallel()

	provs := []providers.Provider{
		testutil.NewMockProvider(),
		testutil.NewMockProvider(),
		testutil.NewMockProvider(),
	}
	s := NewRandomSelector()

	// All available → returns a valid index.
	idx := s.Select(provs, nil)
	require.GreaterOrEqual(t, idx, 0)
	require.Less(t, idx, len(provs))

	// All excluded → -1.
	exclude := map[int]struct{}{0: {}, 1: {}, 2: {}}
	require.Equal(t, -1, s.Select(provs, exclude))

	// Partial exclude → returns a non-excluded index.
	exclude = map[int]struct{}{0: {}, 2: {}}
	idx = s.Select(provs, exclude)
	require.Equal(t, 1, idx)
}

func TestRandomSelector_EmptyList(t *testing.T) {
	t.Parallel()

	s := NewRandomSelector()
	require.Equal(t, -1, s.Select(nil, nil))
	require.Equal(t, -1, s.Select([]providers.Provider{}, nil))
}

func TestRoundRobinSelector_Select(t *testing.T) {
	t.Parallel()

	provs := []providers.Provider{
		testutil.NewMockProvider(),
		testutil.NewMockProvider(),
		testutil.NewMockProvider(),
	}
	s := NewRoundRobinSelector()

	// Sequential calls cycle through all providers.
	require.Equal(t, 0, s.Select(provs, nil))
	require.Equal(t, 1, s.Select(provs, nil))
	require.Equal(t, 2, s.Select(provs, nil))
	require.Equal(t, 0, s.Select(provs, nil))

	// All excluded → -1.
	exclude := map[int]struct{}{0: {}, 1: {}, 2: {}}
	require.Equal(t, -1, s.Select(provs, exclude))
}

func TestRoundRobinSelector_SkipsExcluded(t *testing.T) {
	t.Parallel()

	provs := []providers.Provider{
		testutil.NewMockProvider(),
		testutil.NewMockProvider(),
		testutil.NewMockProvider(),
	}
	s := NewRoundRobinSelector()

	// Exclude index 0 → should pick 1.
	exclude := map[int]struct{}{0: {}}
	require.Equal(t, 1, s.Select(provs, exclude))
	// Next should be 2 (1 was returned last).
	require.Equal(t, 2, s.Select(provs, nil))
	// Then wraps to 0.
	require.Equal(t, 0, s.Select(provs, nil))
}

func TestRoundRobinSelector_EmptyList(t *testing.T) {
	t.Parallel()

	s := NewRoundRobinSelector()
	require.Equal(t, -1, s.Select(nil, nil))
	require.Equal(t, -1, s.Select([]providers.Provider{}, nil))
}

func TestRoundRobinSelector_SingleProvider(t *testing.T) {
	t.Parallel()

	provs := []providers.Provider{testutil.NewMockProvider()}
	s := NewRoundRobinSelector()

	require.Equal(t, 0, s.Select(provs, nil))
	require.Equal(t, 0, s.Select(provs, nil))

	exclude := map[int]struct{}{0: {}}
	require.Equal(t, -1, s.Select(provs, exclude))
}
