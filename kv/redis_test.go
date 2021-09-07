package kv

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"testing"
)

func TestNewRedis(t *testing.T) {
	redisHost := "127.0.0.1:6379"
	redisDb := 0
	redisPass := "hunterhug" // may redis has password
	p, err := NewRedis(
		&MyRedisConf{
			RedisPass:        redisPass,
			RedisDB:          redisDb,
			RedisHost:        redisHost,
			RedisIdleTimeout: 15,
			RedisMaxActive:   15,
			RedisMaxIdle:     15,
		})

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	con := p.Get()
	defer func(con redis.Conn) {
		err := con.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(con)

	do, err := con.Do("SET", "key", []byte("key1"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(do)

	keys, err := redis.ByteSlices(con.Do("KEYS", "*"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, v := range keys {
		fmt.Println(string(v))
	}

}

func TestInitSentinelRedisPool(t *testing.T) {
	redisSentinelHost := "127.0.0.1:26379,127.0.0.1:26380,127.0.0.1:26381"
	redisDb := 0
	redisPass := "hunterhug" // may redis has password
	p, err := NewRedis(
		&MyRedisConf{
			RedisPass:        redisPass,
			RedisDB:          redisDb,
			RedisHost:        redisSentinelHost,
			RedisIdleTimeout: 15,
			RedisMaxActive:   15,
			RedisMaxIdle:     15,
			IsCluster:        true,
			MasterName:       "mymaster",
		})

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	con := p.Get()
	defer func(con redis.Conn) {
		err := con.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(con)

	do, err := con.Do("SET", "key", []byte("key1"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(do)

	keys, err := redis.ByteSlices(con.Do("KEYS", "*"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, v := range keys {
		fmt.Println(string(v))
	}

}
