package session

import (
	"context"
	"sync"
)

type StringSession struct {
	lock     sync.Mutex
	Content  string
	OnChange func(ctx context.Context, content string) error
}

func (s *StringSession) LoadSession(ctx context.Context) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return []byte(s.Content), nil
}

func (s *StringSession) StoreSession(ctx context.Context, data []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Content = string(data)
	if s.OnChange != nil {
		return s.OnChange(ctx, s.Content)
	} else {
		return nil
	}
}
