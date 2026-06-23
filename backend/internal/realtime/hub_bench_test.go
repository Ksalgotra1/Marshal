package realtime

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkConcurrentBroadcastBurst(b *testing.B) {
	tiers := []int{200, 500, 1000}

	for _, n := range tiers {
		b.Run(fmt.Sprintf("clients_%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				hub := NewHub()
				go hub.Run()

				var evictedCount int32
				var wg sync.WaitGroup

				clients := make([]*fakeStream, n)
				dones := make([]chan struct{}, n)

				// Start N clients over a 1.5s window to simulate morning rush
				connectWindow := 1500 * time.Millisecond
				delayBetweenConnects := connectWindow / time.Duration(n)

				for j := 0; j < n; j++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						// Use production buffer size
						client := newFakeStream([]string{"global"}, 256)
						done := make(chan struct{})
						clients[idx] = client
						dones[idx] = done

						// Realistic consumer: slow cellular network simulation (50-300ms delay)
						go func() {
							for {
								select {
								case <-client.send:
									time.Sleep(time.Duration(rand.Intn(250)+50) * time.Millisecond)
								case <-done:
									return
								}
							}
						}()

						hub.RegisterStreamClient(client)
					}(j)
					time.Sleep(delayBetweenConnects)
				}
				wg.Wait()

				// Simulate group-formed events (e.g., 1 per 5s, but we do a short burst here)
				hub.Broadcast("global", GroupFormed("random-group-1", 3, 50))
				time.Sleep(50 * time.Millisecond)
				hub.Broadcast("global", GroupFormed("random-group-2", 3, 50))

				// Wait a moment for any cascading evictions to settle
				time.Sleep(500 * time.Millisecond)

				// Step 1: Check evictions BEFORE any cleanup
				// (otherwise unregistering causes disconnect broadcasts that evict the remaining clients!)
				for j := 0; j < n; j++ {
					if clients[j].isClosed() {
						atomic.AddInt32(&evictedCount, 1)
					}
				}

				// Step 2: Cleanup
				for j := 0; j < n; j++ {
					hub.UnregisterStreamClient(clients[j])
					close(dones[j])
				}

				evictions := atomic.LoadInt32(&evictedCount)
				pct := float64(evictions) / float64(n) * 100.0
				b.Logf("clients=%d evictions=%d (%.1f%%)", n, evictions, pct)
			}
		})
	}
}
