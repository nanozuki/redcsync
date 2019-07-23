package redcsync

import (
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/pkg/errors"
)

func TestMutex(t *testing.T) {
	rs := New(cluster)
	mutex := rs.NewMutex("mutex:{10}:lock")
	if err := mutex.Lock(); err != nil {
		t.Fatal(err)
	}
	if ok, err := mutex.Unlock(); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Error("unlock mutex not ok")
	}
}

func TestMutexExtend(t *testing.T) {
	rs := New(cluster)
	mutex := rs.NewMutex("mutex:{10}:lock")

	err := mutex.Lock()
	if err != nil {
		t.Fatalf("Expected err == nil, got %q", err)
	}
	defer mutex.Unlock()

	time.Sleep(1 * time.Second)

	expiry := getExpirie(cluster, mutex.name)
	ok, err := mutex.Extend()
	if !ok {
		t.Fatalf("Expected ok == true, got %v, err %v", ok, err)
	}
	expiry2 := getExpirie(cluster, mutex.name)

	if expiry2 <= expiry {
		t.Fatalf("Expected expirys > expiry, got %d %d", expiry2, expiry)
	}
}

func TestCompetitionFailure(t *testing.T) {
	rs := New(cluster)
	mutex0 := rs.NewMutex("mutex:{20}:lock")
	mutex1 := rs.NewMutex("mutex:{20}:lock")
	mutex2 := rs.NewMutex("mutex:{21}:lock")

	if err := mutex0.Lock(); err != nil {
		t.Fatal(errors.Wrap(err, "lock mutex0"))
	}
	if ok, err := mutex0.Extend(); !ok {
		t.Fatalf("extend mutex0 failed, err =%v", err)
	}
	if ok, err := mutex0.Extend(); !ok {
		t.Fatalf("extend mutex0 failed, err =%v", err)
	}
	if err := mutex2.Lock(); err != nil {
		t.Fatal(errors.Wrap(err, "lock mutex2"))
	}

	err := mutex1.Lock()
	if err == nil && err != ErrTimeout {
		t.Fatalf("expect mute1.Lock() = ErrTimeout, got %v", err)
	}
}

func getValues(cluster *redisc.Cluster, name string) string {
	conn := cluster.Get()
	conn.Close()
	value, err := redis.String(conn.Do("GET", name))
	if err != nil && err != redis.ErrNil {
		panic(err)
	}
	return value
}

func getExpirie(cluster *redisc.Cluster, name string) int {
	conn := cluster.Get()
	defer conn.Close()
	expiry, err := redis.Int(conn.Do("PTTL", name))
	if err != nil && err != redis.ErrNil {
		panic(err)
	}
	return expiry
}
