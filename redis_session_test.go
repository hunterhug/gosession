package gosession

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/hunterhug/gosession/kv"
	"testing"
)

func debug() redis.Conn {
	redisHost := "127.0.0.1:6379"
	redisDb := 0
	redisPass := "hunterhug" // may redis has password
	p, err := kv.NewRedis(
		&kv.MyRedisConf{
			RedisPass:        redisPass,
			RedisDB:          redisDb,
			RedisHost:        redisHost,
			RedisIdleTimeout: 15,
			RedisMaxActive:   15,
			RedisMaxIdle:     15,
		})

	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	con := p.Get()
	return con
}

func TestWatch(t *testing.T) {
	conn := debug()

	r, err := conn.Do("WATCH", "a")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(r)

	r1, err := conn.Do("SET", "a", "v")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(r1)

	err = conn.Send("MULTI")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = conn.Send("SET", "a", "v")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	r2, err := conn.Do("EXEC")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(r2)

	if r2==nil{
		fmt.Println("no multi ok")
	}
}
