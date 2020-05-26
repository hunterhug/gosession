package kv

import (
	"github.com/gomodule/redigo/redis"
	"strings"
	"time"
)

type MyRedisConf struct {
	RedisHost        string `yaml:"host"`
	RedisMaxIdle     int    `yaml:"max_idle"`
	RedisMaxActive   int    `yaml:"max_active"`
	RedisIdleTimeout int    `yaml:"idle_timeout"`
	RedisDB          int    `yaml:"database"`
	RedisPass        string `yaml:"pass"`
	IsCluster        bool   `yaml:"is_cluster"`
	MasterName       string `yaml:"master_name"`
}

func NewRedis(redisConf *MyRedisConf) (pool *redis.Pool, err error) {
	if redisConf.IsCluster {
		return InitSentinelRedisPool(redisConf)
	}
	pool = &redis.Pool{
		MaxIdle:     redisConf.RedisMaxIdle,
		MaxActive:   redisConf.RedisMaxActive,
		IdleTimeout: time.Duration(redisConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisConf.RedisHost, redis.DialPassword(redisConf.RedisPass), redis.DialDatabase(redisConf.RedisDB))
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
			timeout := 500 * time.Millisecond
			c, err := redis.DialTimeout("tcp", addr, timeout, timeout, timeout)
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
			c, err = redis.Dial("tcp", masterAddr)
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
