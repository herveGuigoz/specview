package catalog

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"specview/internal/project"
)

type Store struct {
	root        *os.Root
	project     project.Project
	scanMu      sync.Mutex
	mu          sync.RWMutex
	current     *Snapshot
	subscribers map[chan uint64]struct{}
}

func New(root *os.Root, p project.Project) (*Store, error) {
	s := &Store{root: root, project: p, subscribers: map[chan uint64]struct{}{}}
	if err := s.Refresh(); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Store) Snapshot() *Snapshot { s.mu.RLock(); defer s.mu.RUnlock(); return s.current }
func (s *Store) Refresh() error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	next, err := Scan(s.root, s.project)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		if s.current == nil {
			return err
		}
		copy := *s.current
		copy.Revision = 0
		copy.Stale = true
		copy.Diagnostics = []string{}
		for _, message := range s.current.Diagnostics {
			if !strings.HasPrefix(message, "Scan failed; previous data retained:") {
				copy.Diagnostics = append(copy.Diagnostics, message)
			}
		}
		copy.Diagnostics = append(copy.Diagnostics, "Scan failed; previous data retained: "+err.Error())
		next = seal(&copy)
	}
	if s.current != nil && s.current.Fingerprint == next.Fingerprint {
		return err
	}
	next.Revision = 1
	if s.current != nil {
		next.Revision = s.current.Revision + 1
	}
	s.current = next
	for ch := range s.subscribers {
		select {
		case <-ch:
		default:
		}
		ch <- next.Revision
	}
	return err
}
func (s *Store) Watch(ctx context.Context, interval time.Duration) {
	timer := time.NewTicker(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.Refresh()
		}
	}
}
func (s *Store) Subscribe() (<-chan uint64, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan uint64, 1)
	s.subscribers[ch] = struct{}{}
	ch <- s.current.Revision
	return ch, func() { s.mu.Lock(); delete(s.subscribers, ch); s.mu.Unlock() }
}
