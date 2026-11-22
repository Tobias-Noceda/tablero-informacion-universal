package mocks

import "sync"

// Steps records what a request does, in order, including its turn on a
// limit (Plans hands out entitlements whose Hold records "hold <limit>
// <holder>" and, on release, "release").
type Steps struct {
	mu   sync.Mutex
	list []string
}

func (s *Steps) Add(step string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.list = append(s.list, step)
}

func (s *Steps) List() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.list...)
}

func (s *Steps) Plans() StaticEntitlements {
	return StaticEntitlements{HoldFn: func(limit, holder string) (func(), error) {
		s.Add("hold " + limit + " " + holder)
		return func() { s.Add("release") }, nil
	}}
}

// In reports whether want happened in that order, maybe with other steps
// between, and the turn was released last.
func (s *Steps) In(want ...string) bool {
	list := s.List()
	at := 0
	for _, step := range list {
		if at < len(want) && step == want[at] {
			at++
		}
	}
	return at == len(want) && len(list) > 0 && list[len(list)-1] == "release"
}
