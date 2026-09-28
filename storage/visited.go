package storage

import "sync"

type VisitedMap struct {
	mu   sync.Mutex
	data map[string]bool
}

func NewVisitedMap() *VisitedMap {
	return &VisitedMap{
		data: make(map[string]bool),
	}
}

// TryVisit возвращает true, если URL еще не был посещен, и помечает его как посещенный.
func (v *VisitedMap) TryVisit(url string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.data[url] {
		return false
	}
	v.data[url] = true
	return true
}
