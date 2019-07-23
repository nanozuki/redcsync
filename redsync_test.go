package redcsync

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
)

var cluster *redisc.Cluster

func createPool(addr string, opts ...redis.DialOption) (*redis.Pool, error) {
	return &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, opts...)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		MaxIdle:     200,
		MaxActive:   200,
		IdleTimeout: 2 * time.Second,
		Wait:        true,
	}, nil
}

func TestMain(m *testing.M) {
	nodesStr := os.Getenv("CLUSTER_NODES")
	cluster = &redisc.Cluster{
		StartupNodes: strings.Split(nodesStr, ","),
		DialOptions: []redis.DialOption{
			redis.DialConnectTimeout(5 * time.Second),
		},
		CreatePool: createPool,
	}
	err := cluster.Refresh()
	if err != nil {
		panic(err)
	}
	result := m.Run()
	cluster.Close()
	os.Exit(result)
}
