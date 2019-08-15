package redcsync

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/pkg/errors"
)

// A DelayFunc is used to decide the amount of time to wait between retries.
type DelayFunc func(tries int) time.Duration

// A Mutex is a distributed mutual exclusion lock.
type Mutex struct {
	name   string
	expiry time.Duration

	tries     int
	delayFunc DelayFunc

	factor float64

	quorum int

	genValueFunc func() (string, error)
	value        string
	until        time.Time

	cluster *redisc.Cluster
}

// Lock locks m. In case it returns an error on failure, you may retry to acquire the lock by calling this method again.
func (m *Mutex) Lock() error {
	value, err := m.genValueFunc()
	if err != nil {
		return err
	}
	m.value = value

	for i := 0; i < m.tries; i++ {
		if i != 0 {
			time.Sleep(m.delayFunc(i))
		}
		ok := m.acquire()
		if ok {
			return nil
		}
	}
	return ErrTimeout
}

// Unlock unlocks m and returns the status of unlock.
func (m *Mutex) Unlock() (bool, error) {
	return m.release()
}

// Extend resets the mutex's expiry and returns the status of expiry extension.
func (m *Mutex) Extend() (bool, error) {
	return m.touch()
}

func genValue() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (m *Mutex) acquire() bool {
	conn := m.cluster.Get()
	defer conn.Close()
	reply, err := redis.String(conn.Do(
		"SET", m.name, m.value, "NX", "PX", int(m.expiry/time.Millisecond),
	))
	return reply == "OK" && err == nil
}

var deleteScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`)

func (m *Mutex) release() (bool, error) {
	conn := m.cluster.Get()
	defer conn.Close()
	if err := redisc.BindConn(conn, m.name); err != nil {
		return false, errors.Wrap(err, "bind conn")
	}
	status, err := redis.Int64(deleteScript.Do(conn, m.name, m.value))
	return status != 0 && err == nil, nil
}

var touchScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("pexpire", KEYS[1], ARGV[2])
	else
		return 0
	end
`)

func (m *Mutex) touch() (bool, error) {
	conn := m.cluster.Get()
	defer conn.Close()
	if err := redisc.BindConn(conn, m.name); err != nil {
		return false, errors.Wrap(err, "bind conn")
	}
	status, err := redis.Int64(touchScript.Do(conn, m.name, m.value, int(m.expiry/time.Millisecond)))
	return status != 0 && err == nil, err
}
