# Distributed Session Implement By Golang

[![GitHub forks](https://img.shields.io/github/forks/hunterhug/gosession.svg?style=social&label=Forks)](https://github.com/hunterhug/gosession/network)
[![GitHub stars](https://img.shields.io/github/stars/hunterhug/gosession.svg?style=social&label=Stars)](https://github.com/hunterhug/gosession/stargazers)
[![GitHub last commit](https://img.shields.io/github/last-commit/hunterhug/gosession.svg)](https://github.com/hunterhug/gosession)
[![Go Report Card](https://goreportcard.com/badge/github.com/hunterhug/gosession)](https://goreportcard.com/report/github.com/hunterhug/gosession)
[![GitHub issues](https://img.shields.io/github/issues/hunterhug/gosession.svg)](https://github.com/hunterhug/gosession/issues)

[中文说明](/README_ZH.md)

Support multi web service share session token, which can keep union state in many diff service.

Now session token can store in:

1. Single mode Redis.
2. Sentinel mode Redis.

## Usage

simple get it by:

```
go get -v github.com/hunterhug/gosession
```

core api:

```go
// token manage
// token will be put in cache database such redis and user info relate with that token will cache too
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

// core user info, it's Id will be the primary key store in cache database such redis
type User struct {
	Id                  string      `json:"id"`     // unique mark
	TokenRemainLiveTime int64       `json:"-"`      // token remain live time in cache
	Detail              interface{} `json:"detail"` // can diy your real user info by config ConfigGetUserInfoFunc()
}
```

example:

```go
package main

import (
	"fmt"
	"github.com/hunterhug/gosession"
	"time"
)

func main() {
	// 1. config redis
	redisHost := "127.0.0.1:6379"
	redisDb := 0
	redisPass := "hunterhug" // may redis has password
	redisConfig := gosession.NewRedisSessionSingleModeConfig(redisHost, redisDb, redisPass)
	// or
	//gosession.NewRedisSessionSentinelModeConfig(":26379,:26380,:26381",0,"mymaster")

	// 2. connect redis session
	tokenManage, err := gosession.NewRedisSession(redisConfig)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 3. config token manage
	tokenManage.ConfigDefaultExpireTime(600)
	tokenManage.ConfigUserKeyPrefix("go-user")
	tokenManage.ConfigTokenKeyPrefix("go-token")
	fn := func(id string) (user *gosession.User, err error) {
		return &gosession.User{
			Id:     id,
			Detail: map[string]string{"detail": id},
		}, nil
	} // get user func diy, you can set it nil
	tokenManage.ConfigGetUserInfoFunc(fn)
	//tokenManage.SetSingleMode()

	// 4. set token
	id := "000001"
	var tokenExpireTimeAlone int64 = 2

	token, err := tokenManage.SetToken(id, tokenExpireTimeAlone)
	if err != nil {
		fmt.Println("set token err:", err.Error())
		return
	}

	fmt.Println("token:", token)

	// can set a lot token
	tokenManage.SetToken(id, 100)
	tokenManage.SetToken(id, 100)
	tokenManage.SetToken(id, 100)

	// 5. list all token
	tokenList, err := tokenManage.ListUserToken(id)
	if err != nil {
		fmt.Println("list token err:", err.Error())
		return
	}
	fmt.Println("list token:", tokenList)

	// 6. check token
	var userExpireTimeAlone int64 = 10 // if ConfigGetUserInfoFunc!=nil, will load user info from func if not exist in redis cache
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

	// 7. sleep to see token is exist?
	time.Sleep(10 * time.Second)
	u, exist, err = tokenManage.CheckTokenOrUpdateUser(token, userExpireTimeAlone)
	if err != nil {
		fmt.Println("sleep check token err:", err.Error())
		return
	}

	fmt.Printf("sleep check token:%#v, %#v,%#v\n", token, u, exist)

	// you can delete all token of one user
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
