package storage

import (
	"sync"
	"testing"
)

func TestVisitedMap(t *testing.T) {
	vm := NewVisitedMap()

	if !vm.TryVisit("https://example.com") {
		t.Error("First visit should return true")
	}

	if vm.TryVisit("https://example.com") {
		t.Error("Second visit should return false")
	}
}

func TestVisitedMap_Concurrent(t *testing.T) {
	vm := NewVisitedMap()
	var wg sync.WaitGroup

	url := "https://concurrent.com"
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if vm.TryVisit(url) {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful visit, got %d", successCount)
	}
}
