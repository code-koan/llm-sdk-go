package fallback

import (
	"math/rand"
	"sync"

	"github.com/code-koan/llm-sdk-go/providers"
)

// Selector picks the next provider from the pool for a given request.
// Implementations must be safe for concurrent use.
type Selector interface {
	// Select returns the index of the next provider to try from the providers
	// list, excluding any indices in exclude. Returns -1 when no provider
	// remains (all have been exhausted).
	Select(providers []providers.Provider, exclude map[int]struct{}) int
}

// RandomSelector picks a random available provider.
type RandomSelector struct {
	mu  sync.Mutex
	rng *rand.Rand
}

// NewRandomSelector creates a RandomSelector with a time-based seed.
func NewRandomSelector() *RandomSelector {
	return &RandomSelector{
		rng: rand.New(rand.NewSource(rand.Int63())),
	}
}

// Select picks a random provider that is not in exclude.
func (s *RandomSelector) Select(providers []providers.Provider, exclude map[int]struct{}) int {
	available := make([]int, 0, len(providers))
	for i := range providers {
		if _, ok := exclude[i]; !ok {
			available = append(available, i)
		}
	}
	if len(available) == 0 {
		return -1
	}

	s.mu.Lock()
	idx := s.rng.Intn(len(available))
	s.mu.Unlock()

	return available[idx]
}

// RoundRobinSelector cycles through providers in order, skipping excluded
// indices across calls.
type RoundRobinSelector struct {
	mu   sync.Mutex
	next int
}

// NewRoundRobinSelector creates a RoundRobinSelector starting at index 0.
func NewRoundRobinSelector() *RoundRobinSelector {
	return &RoundRobinSelector{}
}

// Select returns the next non-excluded provider in round-robin order.
func (s *RoundRobinSelector) Select(providers []providers.Provider, exclude map[int]struct{}) int {
	if len(providers) == 0 {
		return -1
	}

	s.mu.Lock()
	start := s.next % len(providers)
	s.mu.Unlock()

	for i := 0; i < len(providers); i++ {
		idx := (start + i) % len(providers)
		if _, ok := exclude[idx]; !ok {
			s.mu.Lock()
			s.next = idx + 1
			s.mu.Unlock()
			return idx
		}
	}

	return -1
}
