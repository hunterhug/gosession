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
)

var (
	// default prefix of token key
	tokenKeyDefault = "gt"
	// default prefix of user key
	userKeyDefault = "gu"
	// default key expire second
	expireTimeDefault int64 = 3600 * 24 * 7
	// default get user info func
	getUserInfoFuncDefault = func(id string) (*User, error) { return &User{Id: id}, nil }
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
	return &RedisSession{pool: pool, tokenKey: tokenKeyDefault, userKey: userKeyDefault, expireTime: expireTimeDefault, getUserFunc: getUserInfoFuncDefault}, nil
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
	return &RedisSession{pool: pool, tokenKey: tokenKey, userKey: userKey, expireTime: expireTime, getUserFunc: getUserInfoFunc}, nil
}

func NewRedisSessionSingleModeConfig(redisHost string, redisDB int, redisPass string) kv.MyRedisConf {
	return kv.MyRedisConf{
		RedisHost:        redisHost,
		RedisPass:        redisPass,
		RedisDB:          redisDB,
		RedisIdleTimeout: 15,
		RedisMaxActive:   0,
		RedisMaxIdle:     0,
	}
}

func NewRedisSessionSentinelModeConfig(redisHost string, redisDB int, masterName string) kv.MyRedisConf {
	return kv.MyRedisConf{
		RedisHost:        redisHost,
		RedisDB:          redisDB,
		RedisIdleTimeout: 15,
		RedisMaxActive:   0,
		RedisMaxIdle:     0,
		IsCluster:        true,
		MasterName:       masterName,
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
func (s *RedisSession) ConfigExpireTime(second int64) TokenManage {
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
	err = s.set(s.hashTokenKey(token), []byte(userKey), tokenValidTimes)
	if err != nil {
		return
	}

	return token, nil
}

// Refresh token，token expire will be again after some second
func (s *RedisSession) RefreshToken(token string, tokenValidTimes int64) (err error) {
	if token == "" {
		err = errors.New("token empty")
		return
	}
	// simple and rude
	return s.expire(s.hashTokenKey(token), tokenValidTimes)
}

// Delete token when you do action such logout
func (s *RedisSession) DeleteToken(token string) (err error) {
	if token == "" {
		err = errors.New("token empty")
		return
	}
	return s.delete(s.hashTokenKey(token))
}

// Check the token, when cache database exist return user info directly,
// others hit the persistent database and save newest user in cache database then return. such redis check, not check load from mysql.
// you can check user info by that token, if s.getUserFunc == nil do nothing
func (s *RedisSession) CheckTokenOrUpdateUser(token string, userInfoValidTimes int64) (user *User, exist bool, err error) {
	if token == "" {
		err = errors.New("token empty")
		return
	}

	// get user key
	value, ttl, exist, err := s.get(s.hashTokenKey(token))
	if err != nil {
		return nil, false, err
	}

	if !exist {
		return nil, false, nil
	}

	// get user id from user key
	userKey := string(value)
	temp := strings.Split(userKey, "_")
	if len(temp) != 2 || temp[0] != s.userKey {
		return nil, false, errors.New("user key invalid")
	}
	id := temp[1]

	if s.getUserFunc == nil {
		user = new(User)
		user.Id = id
		user.TokenRemainLiveTime = ttl
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
		user.TokenRemainLiveTime = ttl
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
	return user, true, nil
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
	result, exist, err := s.keys(s.userTokenKeys(id))
	if err == nil && exist {
		for _, v := range result {
			otherE := s.delete(v)
			if otherE != nil {
				return otherE
			}
		}
	}
	return
}

// List all token in one user
func (s *RedisSession) ListUserToken(id string) ([]string, error) {
	if id == "" {
		err := errors.New("user id empty")
		return nil, err
	}
	result, _, err := s.keys(s.userTokenKeys(id))
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

// help func to long redis key expire time
func (s *RedisSession) expire(key string, expireSecond int64) (err error) {
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()
	if expireSecond <= 0 {
		expireSecond = s.expireTime
	}
	_, err = conn.Do("EXPIRE", key, expireSecond)
	return err
}

// help func to keys redis
func (s *RedisSession) keys(pattern string) (result []string, exist bool, err error) {
	conn := s.pool.Get()
	if conn.Err() != nil {
		err = conn.Err()
		return
	}

	defer conn.Close()
	keys, err := redis.ByteSlices(conn.Do("KEYS", pattern))
	if err == redis.ErrNil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	result = make([]string, len(keys))
	for k, v := range keys {
		result[k] = string(v)
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

// in order to keys all token of one use
func (s *RedisSession) userTokenKeys(id string) string {
	return fmt.Sprintf("%s_%s_*", s.tokenKey, id)
}
