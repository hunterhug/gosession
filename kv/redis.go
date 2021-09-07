package kv

import (
	"errors"
	"github.com/gomodule/redigo/redis"
	"strings"
	"time"
)

const defaultTimeout = 1

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

	// sentinel
	IsCluster  bool   `yaml:"is_cluster"`
	MasterName string `yaml:"master_name"`

	// timeout, second
	DialConnectTimeout int `yaml:"dial_connect_timeout"`
	DialReadTimeout    int `yaml:"dial_read_timeout"`
	DialWriteTimeout   int `yaml:"dial_write_timeout"`
}

// NewRedis new a redis pool
func NewRedis(redisConf *MyRedisConf) (pool *redis.Pool, err error) {
	if redisConf == nil {
		return nil, errors.New("config nil")
	}

	if redisConf.DialConnectTimeout == 0 {
		redisConf.DialConnectTimeout = defaultTimeout
	}

	if redisConf.DialReadTimeout == 0 {
		redisConf.DialReadTimeout = defaultTimeout
	}

	if redisConf.DialWriteTimeout == 0 {
		redisConf.DialWriteTimeout = defaultTimeout
	}

	if redisConf.IsCluster {
		return initSentinelRedisPool(redisConf)
	}

	idleTimeout := time.Duration(redisConf.RedisIdleTimeout) * time.Second
	dialConnectTimeout := time.Duration(redisConf.DialConnectTimeout) * time.Second
	readTimeout := time.Duration(redisConf.DialReadTimeout) * time.Second
	writeTimeout := time.Duration(redisConf.DialWriteTimeout) * time.Second

	pool = &redis.Pool{
		MaxIdle:     redisConf.RedisMaxIdle,
		MaxActive:   redisConf.RedisMaxActive,
		IdleTimeout: idleTimeout,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisConf.RedisHost,
				redis.DialPassword(redisConf.RedisPass),
				redis.DialDatabase(redisConf.RedisDB),
				redis.DialConnectTimeout(dialConnectTimeout),
				redis.DialReadTimeout(readTimeout),
				redis.DialWriteTimeout(writeTimeout))
			if err != nil {
				return c, err
			}
			return c, nil
		},
	}

	conn := pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
		}
	}(conn)

	_, err = conn.Do("ping")
	return
}

func initSentinelRedisPool(redisConf *MyRedisConf) (pool *redis.Pool, err error) {
	idleTimeout := time.Duration(redisConf.RedisIdleTimeout) * time.Second
	dialConnectTimeout := time.Duration(redisConf.DialConnectTimeout) * time.Second
	readTimeout := time.Duration(redisConf.DialReadTimeout) * time.Second
	writeTimeout := time.Duration(redisConf.DialWriteTimeout) * time.Second

	s := &Sentinel{
		Addrs:      strings.Split(redisConf.RedisHost, ","),
		MasterName: redisConf.MasterName,
		Dial: func(addr string) (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr,
				redis.DialConnectTimeout(dialConnectTimeout),
				redis.DialReadTimeout(readTimeout),
				redis.DialWriteTimeout(writeTimeout))
			if err != nil {
				return c, err
			}
			return c, nil
		},
	}

	pool = &redis.Pool{
		MaxIdle:     redisConf.RedisMaxIdle,
		MaxActive:   redisConf.RedisMaxActive,
		IdleTimeout: idleTimeout,
		Dial: func() (c redis.Conn, err error) {
			masterAddr, err := s.MasterAddr()
			if err != nil {
				return
			}

			// look for master
			c, err = redis.Dial("tcp", masterAddr,
				redis.DialPassword(redisConf.RedisPass),
				redis.DialDatabase(redisConf.RedisDB),
				redis.DialConnectTimeout(dialConnectTimeout),
				redis.DialReadTimeout(readTimeout),
				redis.DialWriteTimeout(writeTimeout))
			if err != nil {
				return c, err
			}

			return c, nil
		},
	}

	conn := pool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
		}
	}(conn)

	_, err = conn.Do("ping")
	return
}
