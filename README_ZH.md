# 分布式 Session Golang库

[![GitHub forks](https://img.shields.io/github/forks/hunterhug/gosession.svg?style=social&label=Forks)](https://github.com/hunterhug/gosession/network)
[![GitHub stars](https://img.shields.io/github/stars/hunterhug/gosession.svg?style=social&label=Stars)](https://github.com/hunterhug/gosession/stargazers)
[![GitHub last commit](https://img.shields.io/github/last-commit/hunterhug/gosession.svg)](https://github.com/hunterhug/gosession)
[![Go Report Card](https://goreportcard.com/badge/github.com/hunterhug/gosession)](https://goreportcard.com/report/github.com/hunterhug/gosession)
[![GitHub issues](https://img.shields.io/github/issues/hunterhug/gosession.svg)](https://github.com/hunterhug/gosession/issues)

[English README](/README_EN.md)
 
支持多个 `Web` 服务共享 `Session` 令牌 `token`，这样可以实现多个服务间共享状态。

现在 Session 令牌可以存储在：

1. 单机模式的 Redis。
2. 哨兵模式的 Redis。

## 如何使用

很简单，执行：

```
go get -v github.com/hunterhug/gosession
```

核心 API:

```go
// 分布式Session管理
// Token
type TokenManage interface {
	SetToken(id string, tokenValidTimes int64) (token string, err error)                               // Set token, expire after some second
	RefreshToken(token string, tokenValidTimes int64) error                                            // Refresh token，token expire will be again after some second
	DeleteToken(token string) error                                                                    // Delete token when you do action such logout
	CheckTokenOrUpdateUser(token string, userInfoValidTimes int64) (user *User, exist bool, err error) // Check the token, when cache database exist return user info directly, others hit the persistent database and save newest user in cache database then return. such redis check, not check load from mysql.
	ListUserToken(id string) ([]string, error)                                                         // List all token of one user
	DeleteUserToken(id string) error                                                                   // Delete all token of this user
	RefreshUser(id []string, userInfoValidTimes int64) error                                           // Refresh cache of user info batch
	DeleteUser(id string) error                                                                        // Delete user info in cache
	AddUser(id string, userInfoValidTimes int64) (user *User, exist bool, err error)                   // Add the user info to cache，expire after some second
	ConfigTokenKeyPrefix(tokenKey string) TokenManage                                                  // Config chain, just cache key prefix
	ConfigUserKeyPrefix(userKey string) TokenManage                                                    // Config chain, just cache key prefix
	ConfigExpireTime(second int64) TokenManage                                                         // Config chain, token expire after second
	ConfigGetUserInfoFunc(fn GetUserInfoFunc) TokenManage                                              // Config chain, when cache not found user info, will load from this func
	SetSingleMode() TokenManage                                                                        // Can set single mode, before one new token gen, will destroy other token
}

// 用户信息，存token在缓存里，比如redis
// 如果有设置ConfigGetUserInfoFunc(fn GetUserInfoFunc)，那么同时也会缓存该用户信息，你可以在函数 type GetUserInfoFunc func(id string) (*User, error) 里将业务用户信息存入 Detail 并返回。
type User struct {
	Id                  string      `json:"id"`     // 用户标志，唯一
	TokenRemainLiveTime int64       `json:"-"`      // token还有多少秒就过期了
	Detail              interface{} `json:"detail"` // 可以存放用户业务信息
}
```

例子：

```go
package main

import (
	"fmt"
	"github.com/hunterhug/gosession"
	"time"
)

func main() {
	// 1. 配置Redis，目前支持单机和哨兵
	redisHost := "127.0.0.1:6379"
	redisDb := 0
	redisPass := "hunterhug" // Redis一般是没有密码的，可以留空
	redisConfig := gosession.NewRedisSessionSingleModeConfig(redisHost, redisDb, redisPass)
	// or
	//gosession.NewRedisSessionSentinelModeConfig()

	// 2. 连接Session管理器
	tokenManage, err := gosession.NewRedisSession(redisConfig)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 3. 配置Session管理器，比如Token 600秒过期，以及token和用户信息key的前缀
	tokenManage.ConfigExpireTime(600)
	tokenManage.ConfigUserKeyPrefix("go-user")
	tokenManage.ConfigTokenKeyPrefix("go-token")
	fn := func(id string) (user *gosession.User, err error) {
		return &gosession.User{
			Id:     id,
			Detail: map[string]string{"detail": id},
		}, nil
	} // 可以设置获取用户信息的函数，如果用户没有缓存，会从该函数加载后存进redis，允许nil
	tokenManage.ConfigGetUserInfoFunc(fn)
	//tokenManage.SetSingleMode() // 你可以设置单点的token

	// 4. 为某用户设置Token
	id := "000001"
	var tokenExpireTimeAlone int64 = 2 // token过期时间设置2秒

	token, err := tokenManage.SetToken(id, tokenExpireTimeAlone)
	if err != nil {
		fmt.Println("set token err:", err.Error())
		return
	}

	fmt.Println("token:", token)

	// 可以设置多个令牌
	tokenManage.SetToken(id, 100)
	tokenManage.SetToken(id, 100)
	tokenManage.SetToken(id, 100)

	// 5. 列出用户所有的令牌
	tokenList, err := tokenManage.ListUserToken(id)
	if err != nil {
		fmt.Println("list token err:", err.Error())
		return
	}
	fmt.Println("list token:", tokenList)

	// 6. 检查token是否存在，存在会返回用户信息
	var userExpireTimeAlone int64 = 10 // 如果用户不存在并且ConfigGetUserInfoFunc!=nil，将会加载用户信息，重新放入redis
	u, exist, err := tokenManage.CheckTokenOrUpdateUser(token, userExpireTimeAlone)
	if err != nil {
		fmt.Println("check token err:", err.Error())
		return
	}

	fmt.Printf("check token:%#v, %#v,%#v\n", token, u, exist)

	err = tokenManage.RefreshToken(token, 5)
	if err != nil {
		fmt.Println("refresh token err:", err.Error())
		return
	}

	u, exist, err = tokenManage.CheckTokenOrUpdateUser(token, userExpireTimeAlone)
	if err != nil {
		fmt.Println("after refresh check token err:", err.Error())
		return
	}

	fmt.Printf("after refresh token:%#v, %#v,%#v\n", token, u, exist)

	// 7. 睡眠一下，看token是不是失效了
	time.Sleep(10 * time.Second)
	u, exist, err = tokenManage.CheckTokenOrUpdateUser(token, userExpireTimeAlone)
	if err != nil {
		fmt.Println("sleep check token err:", err.Error())
		return
	}

	fmt.Printf("sleep check token:%#v, %#v,%#v\n", token, u, exist)

	// 可以删除用户的所有令牌
	tokenList, err = tokenManage.ListUserToken(id)
	if err != nil {
		fmt.Println("sleep list token err:", err.Error())
		return
	}
	fmt.Println("sleep token:", tokenList)

	err = tokenManage.DeleteUserToken(id)
	if err != nil {
		fmt.Println("delete user token err:", err.Error())
		return
	}

	tokenList, err = tokenManage.ListUserToken(id)
	if err != nil {
		fmt.Println("after delete user token list err:", err.Error())
		return
	}
	fmt.Println("after delete user token list:", tokenList)
}
```
