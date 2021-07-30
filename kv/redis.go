package kv

import (
	"github.com/gomodule/redigo/redis"
	"strings"
	"time"
)

// MyRedisConf redis config
type MyRedisConf struct {
	RedisHost string `yaml:"host"`

	// Maximum number of idle connections in the pool.
	RedisMaxIdle int `yaml:"max_idle"`

	// Maximum number of connections allocated by the pool at a given time.
	// When zero, there is no limit on the number of connections in the pool.
	RedisMaxActive int `yaml:"max_active"`

	// Close connections after remaining idle for this duration. If the value
	// is zero, then idle connections are not closed. Applications should set
	// the timeout to a value less than the server's timeout.
	RedisIdleTimeout int    `yaml:"idle_timeout"`
	RedisDB          int    `yaml:"database"`
	RedisPass        string `yaml:"pass"`
	IsCluster        bool   `yaml:"is_cluster"`  // sentinel
	MasterName       string `yaml:"master_name"` // sentinel
}

// NewRedis new a redis pool
func NewRedis(redisConf *MyRedisConf) (pool *redis.Pool, err error) {
	// sentinel use other func
	if redisConf.IsCluster {
		return InitSentinelRedisPool(redisConf)
	}
	pool = &redis.Pool{
		MaxIdle:     redisConf.RedisMaxIdle,
		MaxActive:   redisConf.RedisMaxActive,
		IdleTimeout: time.Duration(redisConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			timeout := 500 * time.Millisecond
			c, err := redis.Dial("tcp", redisConf.RedisHost, redis.DialPassword(redisConf.RedisPass), redis.DialDatabase(redisConf.RedisDB), redis.DialConnectTimeout(timeout),
				redis.DialReadTimeout(timeout), redis.DialWriteTimeout(timeout))
			if err != nil {
				return c, err
			}
			return c, nil
		},
	}

	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("ping")
	return
}

func InitSentinelRedisPool(redisConf *MyRedisConf) (pool *redis.Pool, err error) {
	s := &Sentinel{
		Addrs:      strings.Split(redisConf.RedisHost, ","),
		MasterName: redisConf.MasterName,
		Dial: func(addr string) (redis.Conn, error) {
			timeout := 1000 * time.Millisecond
			c, err := redis.Dial("tcp", addr, redis.DialConnectTimeout(timeout),
				redis.DialReadTimeout(timeout), redis.DialWriteTimeout(timeout))
			if err != nil {
				return c, err
			}
			return c, nil
		},
	}

	pool = &redis.Pool{
		MaxIdle:     redisConf.RedisMaxIdle,
		MaxActive:   redisConf.RedisMaxActive,
		IdleTimeout: time.Duration(redisConf.RedisIdleTimeout) * time.Second,
		Dial: func() (c redis.Conn, err error) {
			masterAddr, err := s.MasterAddr()
			if err != nil {
				return
			}

			timeout := 1000 * time.Millisecond

			// look for master
			c, err = redis.Dial("tcp", masterAddr, redis.DialPassword(redisConf.RedisPass), redis.DialConnectTimeout(timeout),
				redis.DialReadTimeout(timeout), redis.DialWriteTimeout(timeout))
			if err != nil {
				return c, err
			}
			c.Do("SELECT", redisConf.RedisDB)
			return c, nil
		},
	}

	conn := pool.Get()
	defer conn.Close()

	_, err = conn.Do("ping")
	return
}
