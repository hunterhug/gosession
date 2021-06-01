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
	redisPass := "root" // may redis has password
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
