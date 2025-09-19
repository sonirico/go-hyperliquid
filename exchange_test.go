package hyperliquid

import (
	"sync"
	"testing"
	"time"

	"testing/synctest"
)

func TestNextNonce(t *testing.T) {
	cases := []struct {
		name          string
		preLastOffset int64         // preLast = baseNow + preLastOffset
		advance       time.Duration // fake time to advance before calling nextNonce
	}{
		{
			name:          "fresh (last < now) uses now",
			preLastOffset: -50, // last is behind the clock
			advance:       0,
		},
		{
			name:          "same millisecond increments",
			preLastOffset: 0, // last == now
			advance:       0,
		},
		{
			name:          "clock behind last increments",
			preLastOffset: 100, // last > now
			advance:       0,
		},
		{
			name:          "after time advances uses new now",
			preLastOffset: 0,                     // equal first
			advance:       10 * time.Millisecond, // then clock moves forward
		},
	}

	for _, tc := range cases {
		// Each table case runs in its own bubble.
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var e Exchange

				baseNow := time.Now().UnixMilli() // fake time start of this bubble
				preLast := baseNow + tc.preLastOffset
				e.lastNonce.Store(preLast)

				if tc.advance > 0 {
					time.Sleep(tc.advance) // advances fake clock instantly within the bubble
				}

				got := e.nextNonce()

				now := baseNow + tc.advance.Milliseconds()
				want := now
				if now <= preLast {
					want = preLast + 1
				}

				if got != want {
					t.Fatalf(
						"nextNonce()=%d, want %d (preLast=%d now=%d advance=%s)",
						got,
						want,
						preLast,
						now,
						tc.advance,
					)
				}
				if stored := e.lastNonce.Load(); stored != want {
					t.Fatalf("lastNonce=%d, want %d", stored, want)
				}
			})
		})
	}
}

func TestNextNonce_SequentialMonotonicity(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var e Exchange

		base := time.Now().UnixMilli()

		// Call multiple times without advancing fake time: must strictly increase by 1 each call.
		const n = 5
		vals := make([]int64, 0, n)
		for i := 0; i < n; i++ {
			vals = append(vals, e.nextNonce())
		}
		for i := 0; i < n; i++ {
			want := base + int64(i)
			if vals[i] != want {
				t.Fatalf("seq[%d]=%d, want %d", i, vals[i], want)
			}
		}

		// Advance 7ms; next should jump to base+7
		time.Sleep(7 * time.Millisecond)
		next := e.nextNonce()
		if next != base+7 {
			t.Fatalf("after advance got %d, want %d", next, base+7)
		}
	})
}

func TestNextNonce_ConcurrencyUniqueness(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var e Exchange
		base := time.Now().UnixMilli()

		const N = 1000
		results := make([]int64, N)
		var wg sync.WaitGroup
		wg.Add(N)
		for i := 0; i < N; i++ {
			go func(i int) {
				defer wg.Done()
				results[i] = e.nextNonce()
			}(i)
		}
		wg.Wait()

		seen := make(map[int64]struct{}, N)
		var min, max int64
		for i, v := range results {
			if _, dup := seen[v]; dup {
				t.Fatalf("duplicate nonce: %d", v)
			}
			seen[v] = struct{}{}
			if i == 0 || v < min {
				min = v
			}
			if i == 0 || v > max {
				max = v
			}
		}

		if min != base {
			t.Fatalf("min=%d want %d", min, base)
		}
		if max != base+N-1 {
			t.Fatalf("max=%d want %d", max, base+N-1)
		}
	})
}
