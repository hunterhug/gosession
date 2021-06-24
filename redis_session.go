/*
	All right reserved：https://github.com/hunterhug/gosession at 2020
	Attribution-NonCommercial-NoDerivatives 4.0 International
	You can use it for education only but can't make profits for any companies and individuals!
*/
package gosession

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/hunterhug/gosession/kv"
	"strings"
	"time"
)

var (
	// default prefix of token key
	tokenKeyDefault = "go-t"
	// default prefix of user key
	userKeyDefault = "go-u"
	// default key expire second
	expireTimeDefault int64 = 3600 * 24 * 7
	// default get user info func
	getUserInfoFuncDefault = func(id string) (*User, error) { return &User{Id: id}, nil }

	// list all token, hash key expire
	TokenMapKeyExpireTime int64 = 3600 * 24 * 30
)

// func get user info from where
type GetUserInfoFunc func(id string) (*User, error)

// session by redis
type RedisSession struct {
	pool         *redis.Pool                    // redis pool can single mode or other mode
	getUserFunc  func(id string) (*User, error) // when not hit cache will get user from this func
	tokenKey     string                         // prefix of token，default 'got'
	userKey      string                         // prefix of user info cache ，default 'gou'
	expireTime   int64                          // token expire how much second，default  7 days
	isSingleMode bool                           // is single token, new token will destroy other token
}

// new a redis session
func NewRedisSession(redisConf kv.MyRedisConf) (TokenManage, error) {
	pool, err := kv.NewRedis(&redisConf)
	if err != nil {
		return nil, err
	}
	return &RedisSession{pool: pool, tokenKey: tokenKeyDefault, userKey: userKeyDefault, expireTime: expireTimeDefault, getUserFunc: nil}, nil
}

// new a redis session by redis pool
func NewRedisSessionWithPool(pool *redis.Pool) TokenManage {
	return &RedisSession{pool: pool, tokenKey: tokenKeyDefault, userKey: userKeyDefault, expireTime: expireTimeDefault, getUserFunc: nil}
}

// new a redis session, config all
// define prefix of token and user key
func NewRedisSessionAll(redisConf kv.MyRedisConf, tokenKey, userKey string, expireTime int64, getUserInfoFunc GetUserInfoFunc) (TokenManage, error) {
	pool, err := kv.NewRedis(&redisConf)
	if err != nil {
		return nil, err
	}

	tokenKey = strings.Replace(tokenKey, "_", "-", -1)
	userKey = strings.Replace(userKey, "_", "-", -1)

	if expireTime <= 0 {
		expireTime = expireTimeDefault
	}
	return &RedisSession{pool: pool, tokenKey: tokenKey, userKey: userKey, expireTime: expireTime, getUserFunc: getUserInfoFunc}, nil
}

// redis single mode config
func NewRedisSessionSingleModeConfig(redisHost string, redisDB int, redisPass string) kv.MyRedisConf {
	return kv.MyRedisConf{
		RedisHost:        redisHost,
		RedisPass:        redisPass,
		RedisDB:          redisDB,
		RedisIdleTimeout: 15,
		RedisMaxActive:   20,
		RedisMaxIdle:     30,
	}
}

// redis sentinel mode config
// redisHost is sentinel address, not redis address
// redisPass is redis password
func NewRedisSessionSentinelModeConfig(redisHost string, redisDB int, redisPass string, masterName string) kv.MyRedisConf {
	return kv.MyRedisConf{
		RedisHost:        redisHost,
		RedisDB:          redisDB,
		RedisIdleTimeout: 15,
		RedisMaxActive:   20,
		RedisMaxIdle:     30,
		IsCluster:        true,
		MasterName:       masterName,
		RedisPass:        redisPass,
	}
}

// config by chain
func (s *RedisSession) ConfigTokenKeyPrefix(tokenKey string) TokenManage {
	tokenKey = strings.Replace(tokenKey, "_", "-", -1)
	s.tokenKey = tokenKey
	return s
}

// config by chain
func (s *RedisSession) ConfigUserKeyPrefix(userKey string) TokenManage {
	userKey = strings.Replace(userKey, "_", "-", -1)
	s.userKey = userKey
	return s
}

// config by chain
func (s *RedisSession) ConfigDefaultExpireTime(second int64) TokenManage {
	if second <= 0 {
		return s
	}
	s.expireTime = second
	return s
}

// config by chain
func (s *RedisSession) ConfigGetUserInfoFunc(fn GetUserInfoFunc) TokenManage {
	if fn == nil {
		return s
	}
	s.getUserFunc = fn
	return s
}

// set single mode, new token will destroy other token
func (s *RedisSession) SetSingleMode() TokenManage {
	s.isSingleMode = true
	return s
}

// Set token, expire after some second
func (s *RedisSession) SetToken(id string, tokenValidTimes int64) (token string, err error) {
	// user id can not nil
	if id == "" {
		err = errors.New("user id nil")
		return
	}

	// gen token by user id, every time will gen new
	token = s.genToken(id)

	// gen user key by user id
	userKey := s.hashUserKey(id)

	// if single, destroy other token
	if s.isSingleMode {
		err = s.DeleteUserToken(id)
		if err != nil {
			return
		}
	}

	// relate token and user in redis
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()
	err = conn.Send("MULTI")
	if err != nil {
		return
	}

	if tokenValidTimes <= 0 {
		tokenValidTimes = s.expireTime
	}

	err = conn.Send("SETEX", s.hashTokenKey(token), tokenValidTimes, []byte(userKey))
	if err != nil {
		return
	}

	tokenMapKey := s.userTokenMapKey(id)
	err = conn.Send("HSET", tokenMapKey, token, time.Now().Unix()+tokenValidTimes)
	if err != nil {
		return
	}

	err = conn.Send("EXPIRE", tokenMapKey, TokenMapKeyExpireTime)
	if err != nil {
		return
	}

	_, err = conn.Do("EXEC")
	return token, nil
}

// Refresh token，token expire will be again after some second
func (s *RedisSession) RefreshToken(token string, tokenValidTimes int64) (err error) {
	if token == "" {
		err = errors.New("token empty")
		return
	}

	temp := strings.Split(token, "_")
	if len(temp) < 2 || temp[0] == "" {
		err = errors.New("token wrong")
		return
	}

	id := temp[0]

	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()
	err = conn.Send("MULTI")
	if err != nil {
		return
	}

	if tokenValidTimes <= 0 {
		tokenValidTimes = s.expireTime
	}

	err = conn.Send("EXPIRE", s.hashTokenKey(token), tokenValidTimes)
	if err != nil {
		return
	}

	tokenMapKey := s.userTokenMapKey(id)

	err = conn.Send("HSET", tokenMapKey, token, time.Now().Unix()+tokenValidTimes)
	if err != nil {
		return
	}

	err = conn.Send("EXPIRE", tokenMapKey, TokenMapKeyExpireTime)
	if err != nil {
		return
	}

	_, err = conn.Do("EXEC")
	if err != nil {
		return
	}

	return
}

// Delete token when you do action such logout
func (s *RedisSession) DeleteToken(token string) (err error) {
	if token == "" {
		err = errors.New("token empty")
		return
	}

	temp := strings.Split(token, "_")
	if len(temp) < 2 || temp[0] == "" {
		err = errors.New("token wrong")
		return
	}

	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()

	return s.deleteToken(conn, temp[0], token)
}

func (s *RedisSession) deleteToken(conn redis.Conn, id string, token string) (err error) {
	if token == "" {
		err = errors.New("token empty")
		return
	}

	if id == "" {
		temp := strings.Split(token, "_")
		if len(temp) < 2 || temp[0] == "" {
			err = errors.New("token wrong")
			return
		}

		id = temp[0]
	}

	err = conn.Send("MULTI")
	if err != nil {
		return err
	}

	err = conn.Send("DEL", s.hashTokenKey(token))
	if err != nil {
		return err
	}

	err = conn.Send("HDEL", s.userTokenMapKey(id), token)
	if err != nil {
		return err
	}

	_, err = conn.Do("EXEC")
	return err
}

// Check the token, when cache database exist return user info directly,
// others hit the persistent database and save newest user in cache database then return. such redis check, not check load from mysql.
// you can check user info by that token, if s.getUserFunc == nil do nothing
func (s *RedisSession) CheckTokenOrUpdateUser(token string, userInfoValidTimes int64) (user *User, exist bool, err error) {
	if token == "" {
		err = errors.New("token empty")
		return
	}

	temp := strings.Split(token, "_")
	if len(temp) < 2 || temp[0] == "" {
		err = errors.New("token wrong")
		return
	}

	id := temp[0]

	// get user key
	value, ttl, exist, err := s.get(s.hashTokenKey(token))
	if err != nil {
		return nil, false, err
	}

	tokenMapKey := s.userTokenMapKey(id)

	if !exist || ttl <= 1 {
		err = s.deleteMap(tokenMapKey, token)
		if err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}

	expireTime, exist, err := s.hGet(tokenMapKey, token)
	if err != nil {
		return nil, false, err
	}

	// get user id from user key
	userKey := string(value)
	temp = strings.Split(userKey, "_")
	if len(temp) != 2 || temp[0] != s.userKey || temp[1] != id {
		return nil, false, errors.New("user key invalid")
	}

	if s.getUserFunc == nil || userInfoValidTimes < 0 {
		user = new(User)
		user.Id = id
		user.TokenRemainLiveTime = ttl
		user.Token = token
		user.TokenExpireTime = expireTime
		return user, true, nil
	}

	// get user info by user key
	value, _, exist, err = s.get(userKey)
	if err != nil {
		return nil, false, err
	}

	// when exit user info return directly
	user = new(User)
	if exist {
		err = json.Unmarshal(value, user)
		if err != nil {
			return nil, false, err
		}
		user.Id = id
		user.TokenRemainLiveTime = ttl
		user.Token = token
		user.TokenExpireTime = expireTime
		return user, true, nil
	}

	// load user and add into cache
	user, exist, err = s.AddUser(id, userInfoValidTimes)
	if err != nil {
		return nil, false, err
	}

	if !exist {
		return nil, false, nil
	}

	user.TokenRemainLiveTime = ttl
	user.Token = token
	user.TokenExpireTime = expireTime
	return user, true, nil
}

func (s *RedisSession) CheckToken(token string) (user *User, exist bool, err error) {
	return s.CheckTokenOrUpdateUser(token, -1)
}

// Add the user info to cache，expire after some second
func (s *RedisSession) AddUser(id string, userInfoValidTimes int64) (user *User, exist bool, err error) {
	if s.getUserFunc == nil {
		return nil, false, errors.New("getUserFunc nil")
	}

	if id == "" {
		err = errors.New("user id empty")
		return
	}

	// get user info from outer func
	user, err = s.getUserFunc(id)
	if err != nil {
		return nil, false, err
	}

	if user == nil {
		user = new(User)
		user.Id = id
	}

	// gen user key
	userKey := s.hashUserKey(user.Id)

	// get user info raw
	raw, err := json.Marshal(user)
	if err != nil {
		return nil, false, err
	}

	// set into redis
	err = s.set(userKey, raw, userInfoValidTimes)
	if err != nil {
		return nil, false, err
	}

	user.Id = id
	return user, true, nil
}

// Refresh cache of user info batch
func (s *RedisSession) RefreshUser(ids []string, userInfoValidTimes int64) (err error) {
	// very rude
	for _, id := range ids {
		_, _, err = s.AddUser(id, userInfoValidTimes)
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete all token of this user
func (s *RedisSession) DeleteUserToken(id string) (err error) {
	if id == "" {
		err = errors.New("user id empty")
		return
	}

	tokenMapKey := s.userTokenMapKey(id)

	result, exist, err := s.tokenKeys(tokenMapKey)
	if err != nil {
		return err
	}

	if exist && len(result) > 0 {

		conn := s.pool.Get()
		if conn.Err() != nil {
			err = conn.Err()
			return
		}

		defer conn.Close()

		err = conn.Send("MULTI")
		if err != nil {
			return err
		}

		for _, v := range result {
			err = conn.Send("DEL", s.hashTokenKey(v))
			if err != nil {
				return err
			}

			err = conn.Send("HDEL", tokenMapKey, v)
			if err != nil {
				return err
			}
		}

		_, err = conn.Do("EXEC")
		return err
	}

	return
}

// List all token in one user
func (s *RedisSession) ListUserToken(id string) ([]string, error) {
	if id == "" {
		err := errors.New("user id empty")
		return nil, err
	}
	result, _, err := s.tokenKeys(s.userTokenMapKey(id))
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Delete user info in cache
func (s *RedisSession) DeleteUser(id string) (err error) {
	if id == "" {
		err = errors.New("user id empty")
		return
	}
	return s.delete(s.hashUserKey(id))
}

// help func to set redis key which use MULTI order
func (s *RedisSession) set(key string, value []byte, expireSecond int64) (err error) {
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()
	err = conn.Send("MULTI")
	if err != nil {
		return err
	}

	err = conn.Send("SET", key, value)
	if err != nil {
		return err
	}

	// when expireSecond not large 0 will use default second
	if expireSecond <= 0 {
		expireSecond = s.expireTime
	}

	err = conn.Send("EXPIRE", key, expireSecond)
	if err != nil {
		return err
	}

	_, err = conn.Do("EXEC")
	return
}

// help func to delete redis key
func (s *RedisSession) delete(key string) (err error) {
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()
	_, err = conn.Do("DEL", key)
	return err
}

func (s *RedisSession) deleteMap(key, subKey string) (err error) {
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()
	_, err = conn.Do("HDEL", key, subKey)
	return err
}

func (s *RedisSession) deleteMapWithConn(conn redis.Conn, key, subKey string) (err error) {
	_, err = conn.Do("HDEL", key, subKey)
	return err
}

func (s *RedisSession) tokenKeys(pattern string) (result []string, exist bool, err error) {
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()

	keys, err := redis.StringMap(conn.Do("HGETALL", pattern))
	if err == redis.ErrNil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	result = make([]string, len(keys))
	for k, v := range keys {

		if SI(v) <= time.Now().Unix() {
			err = s.deleteMapWithConn(conn, pattern, k)
			if err != nil {
				return nil, false, err
			}

			continue
		} else {
		}
		result = append(result, k)
	}
	return result, true, nil
}

// help func to get redis key
func (s *RedisSession) get(key string) (value []byte, ttl int64, exist bool, err error) {
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	value, err = redis.Bytes(conn.Do("GET", key))
	if err == redis.ErrNil {
		return nil, 0, false, nil
	} else if err != nil {
		return nil, 0, false, err
	}

	ttl, err = redis.Int64(conn.Do("TTL", key))
	if err == redis.ErrNil {
		return nil, 0, false, nil
	} else if err != nil {
		return nil, 0, false, err
	}

	return value, ttl, true, nil
}

// help func to hGet redis key
func (s *RedisSession) hGet(key, subKey string) (value int64, exist bool, err error) {
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	value, err = redis.Int64(conn.Do("HGET", key, subKey))
	if err == redis.ErrNil {
		return 0, false, nil
	} else if err != nil {
		return 0, false, err
	}

	return value, true, nil
}

// gen token, will random gen string
func (s *RedisSession) genToken(id string) string {
	// has prefix user id
	return fmt.Sprintf("%s_%s", id, GetGUID())
}

// gen hashTokenKey, as a key in redis, it's value will be hashUserKey
func (s *RedisSession) hashTokenKey(token string) string {
	return fmt.Sprintf("%s_%s", s.tokenKey, token)
}

// gen hashUserKey, as a key in redis, it's value will be user info
func (s *RedisSession) hashUserKey(id string) string {
	return fmt.Sprintf("%s_%s", s.userKey, id)
}

// hash map key which struct store all token
func (s *RedisSession) userTokenMapKey(id string) string {
	return fmt.Sprintf("%s_%s", s.tokenKey, id)
}
