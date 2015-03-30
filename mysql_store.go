package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// require parseTime=true if use with github.com/go-sql-driver/mysql
// Schema:
// CREATE TABLE `session` (
//   `id` VARCHAR(40) NOT NULL,
//   `expires_at` DATETIME NOT NULL,
//   `data` VARCHAR(45) NOT NULL,
//   PRIMARY KEY (`id`));
type MySQLStore struct {
	db        *sql.DB
	maxAge    time.Duration
	gcTimer   *time.Timer
	closeChan chan bool
}

func NewMySQLStore(db *sql.DB, maxAge int) *MySQLStore {
	duration := time.Second * time.Duration(maxAge)
	store := &MySQLStore{
		db:        db,
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
				if _, err := db.Exec("DELETE FROM session WHERE expires_at < ?", time.Now()); err != nil {
					log.Printf("[session.MySQLStore] clean up error: %s\n", err)
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

func (s *MySQLStore) Load(id string) (*Session, error) {
	row := s.db.QueryRow("SELECT expires_at, data FROM session WHERE id = ?", id)
	var expiresAt time.Time
	var sess *Session = nil
	datajson := ""
	err := row.Scan(&expiresAt, &datajson)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, fmt.Errorf("[session.MySQLStore] get: %s", err)
		}
	} else {
		sess = NewSession(id)
		sess.expiresAt = expiresAt
		if err = json.Unmarshal([]byte(datajson), &sess.data); err != nil {
			return nil, fmt.Errorf("[session.MySQLStore] get: %s", err)
		}
	}
	return sess, nil
}

func (s *MySQLStore) Has(id string) (bool, error) {
	row := s.db.QueryRow("SELECT COUNT(*) FROM session WHERE id = ?", id)
	exists := false
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("[session.MySQLStore] has: %s", err)
	}
	return exists, nil
}

func (s *MySQLStore) Revoke(id string) error {
	if _, err := s.db.Exec("DELETE FROM session WHERE id = ? LIMIT 1", id); err != nil {
		return err
	}
	return nil
}

func (s *MySQLStore) Renew(session *Session) error {
	return s.Save(session)
}

func (s *MySQLStore) Save(session *Session) error {
	data, err := json.Marshal(session.data)
	if err != nil {
		return fmt.Errorf("[session.MySQLStore] marshal error: %s", err)
	}
	datastr := string(data)
	expiry := time.Now().Add(s.maxAge)
	session.expiresAt = expiry
	sql := `INSERT INTO session(id, expires_at, data) VALUES(?, ?, ?) ON DUPLICATE KEY UPDATE data = ?`
	_, err = s.db.Exec(sql, session.id, expiry, datastr, datastr)
	if err != nil {
		return fmt.Errorf("[session.MySQLStore] update error: %s", err)
	}
	return nil
}

func (s *MySQLStore) Close() error {
	s.gcTimer.Stop()
	s.closeChan <- true
	<-s.closeChan
	//fmt.Println("Closed")
	return nil
}
