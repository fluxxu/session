package session

// For dev only

import (
	"sync"
	"time"
)

type MemoryStore struct {
	maxAge    time.Duration
	mutex     sync.Mutex
	storage   map[string]*Session
	gcTimer   *time.Timer
	closeChan chan bool
}

func NewMemoryStore(maxAge int) *MemoryStore {
	duration := time.Second * time.Duration(maxAge)
	store := &MemoryStore{
		maxAge:    duration,
		storage:   make(map[string]*Session),
		gcTimer:   time.NewTimer(duration),
		closeChan: make(chan bool),
	}

	go func() {
		//fmt.Println("gc enter")
	loop:
		for {
			select {
			case <-store.gcTimer.C:
				//fmt.Println("gc...")
				store.mutex.Lock()
				now := time.Now()
				for id, sess := range store.storage {
					if sess.expiresAt.Before(now) {
						//fmt.Println("Delete: ", id)
						delete(store.storage, id)
					}
				}
				store.mutex.Unlock()
				store.gcTimer.Reset(duration)
			case <-store.closeChan:
				store.closeChan <- true
				break loop
			}
		}
		//fmt.Println("gc exit")
	}()

	return store
}

func (s *MemoryStore) Load(id string) (*Session, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	sess, ok := s.storage[id]
	if !ok {
		sess = nil
	}
	return sess, nil
}

func (s *MemoryStore) Has(id string) (bool, error) {
	_, has := s.storage[id]
	return has, nil
}

func (s *MemoryStore) Revoke(id string) error {
	s.mutex.Lock()
	delete(s.storage, id)
	s.mutex.Unlock()
	return nil
}

func (s *MemoryStore) Renew(session *Session) error {
	session.expiresAt = time.Now().Add(s.maxAge)
	return nil
}

func (s *MemoryStore) Save(session *Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session.expiresAt = time.Now().Add(s.maxAge)
	s.storage[session.id] = session
	return nil
}

func (s *MemoryStore) Close() error {
	s.gcTimer.Stop()
	s.closeChan <- true
	<-s.closeChan
	//fmt.Println("Closed")
	return nil
}
