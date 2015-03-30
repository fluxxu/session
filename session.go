package session

import (
	_ "net/http"
	"sync"
	"time"
)

type Session struct {
	id        string
	store     Store
	mutex     sync.Mutex
	data      map[string]interface{}
	expiresAt time.Time
}

func (s *Session) Id() string {
	return s.id
}

func (s *Session) Has(key string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.data[key]
	return ok
}

func (s *Session) Get(key string) interface{} {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.data[key]
}

func (s *Session) Set(key string, value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.data[key] = value
}

func (s *Session) Unset(key string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.data, key)
}

func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}

func NewSession(id string) *Session {
	return &Session{
		id:   id,
		data: make(map[string]interface{}),
	}
}

type Store interface {
	// Check session existence
	Has(id string) (bool, error)

	// Load a session by id, if not found, return nil
	Load(id string) (*Session, error)

	// Delete a session
	Revoke(id string) error

	// Extend session expiry
	Renew(session *Session) error

	// Save a session. Session expiry gets updated too
	Save(session *Session) error

	// Clean up
	Close() error
}
