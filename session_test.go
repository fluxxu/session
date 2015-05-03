package session

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2"
	"testing"
	"time"
)

var stores map[string]Store

func TestGetHasExpire(t *testing.T) {
	for name, store := range stores {
		sess := NewSession("1")
		err := store.Save(sess)
		assert.NoError(t, err, name)

		has, err := store.Has("1")
		assert.NoError(t, err, name)
		assert.Equal(t, true, has, name)
	}

	time.Sleep(time.Second * 2)

	for name, store := range stores {
		has, err := store.Has("1")
		assert.NoError(t, err, name)
		assert.Equal(t, false, has, name)
	}
}

func TestRevoke(t *testing.T) {
	for name, store := range stores {
		sess := NewSession("revoke")
		assert.NoError(t, store.Save(sess), name)

		has, err := store.Has("revoke")
		assert.NoError(t, err, name)
		assert.Equal(t, true, has, name)

		err = store.Revoke("revoke")
		assert.NoError(t, err, name)

		has, err = store.Has("revoke")
		assert.NoError(t, err, name)
		assert.Equal(t, false, has, name)
	}
}

func TestSessionSetGetUnsetHas(t *testing.T) {
	for name, store := range stores {
		sess := NewSession("getset")
		sess.Set("value", 123)
		assert.NoError(t, store.Save(sess), name)

		assert.Equal(t, sess.Get("value"), 123)

		assert.NoError(t, store.Save(sess), name)

		sess, err := store.Load("getset")
		assert.NoError(t, err, name)
		assert.Equal(t, sess.Get("value"), 123)

		sess.Set("value", 456)
		sess.Set("value2", 789)

		assert.NoError(t, store.Save(sess), name)

		sess, err = store.Load("getset")
		assert.NoError(t, err, name)
		assert.Equal(t, sess.Get("value"), 456, name)
		assert.Equal(t, sess.Get("value2"), 789, name)

		sess.Unset("value2")
		assert.Equal(t, nil, sess.Get("value2"), name)

		assert.Equal(t, false, sess.Has("value2"), name)
	}
}

func init() {
	db, err := sql.Open("mysql", "root@/session_test?parseTime=true")
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}

	sess, err := mgo.Dial("mongo")
	if err != nil {
		panic(err)
	}

	c := sess.DB("test").C("sessions")

	stores = map[string]Store{
		"MemoryStore": NewMemoryStore(1),
		"MySQLStore":  NewMySQLStore(db, 1),
		"MongoStore":  NewMongoStore(c, 1),
	}
}
