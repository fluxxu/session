package session

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

type MongoStore struct {
	c         *mgo.Collection
	maxAge    time.Duration
	gcTimer   *time.Timer
	closeChan chan bool
}

func NewMongoStore(c *mgo.Collection, maxAge int) *MongoStore {
	duration := time.Second * time.Duration(maxAge)
	store := &MongoStore{
		c:         c,
		maxAge:    duration,
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
				s := bson.M{
					"expires_at": bson.M{
						"$lt": time.Now(),
					},
				}
				if err := c.Remove(s); err != nil {
					log.Printf("[session.MongoStore] clean up error: %s\n", err)
				}
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

func (s *MongoStore) Load(id string) (*Session, error) {
	doc := struct {
		ExpiresAt time.Time
		Data      map[string]interface{}
	}{}
	err := s.c.Find(bson.M{"id": id}).One(&doc)
	if err != nil {
		return nil, fmt.Errorf("[session.MongoStore] get: %s", err)
	}
	sess := NewSession(id)
	sess.expiresAt = doc.ExpiresAt
	sess.data = doc.Data
	return sess, nil
}

func (s *MongoStore) Has(id string) (bool, error) {
	n, err := s.c.Find(bson.M{"id": id}).Count()
	if err != nil {
		return false, fmt.Errorf("[session.MongoStore] count: %s", err)
	}
	return n > 0, nil
}

func (s *MongoStore) Revoke(id string) error {
	err := s.c.Remove(bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("[session.MongoStore] remove: %s", err)
	}
	return nil
}

func (s *MongoStore) Renew(session *Session) error {
	return s.Save(session)
}

func (s *MongoStore) Save(session *Session) error {
	expiry := time.Now().Add(s.maxAge)
	session.expiresAt = expiry
	_, err := s.c.Upsert(bson.M{"id": session.id}, bson.M{
		"$set": bson.M{
			"expires_at": expiry,
			"data":       session.data,
		},
		"$setOnInsert": bson.M{
			"id": session.id,
		},
	})
	return err
}

func (s *MongoStore) Close() error {
	s.gcTimer.Stop()
	s.closeChan <- true
	<-s.closeChan
	//fmt.Println("Closed")
	return nil
}
