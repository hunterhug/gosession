package main

import (
	"fmt"
	"github.com/hunterhug/gosession"
)

func main() {
	tokenManage, err := gosession.NewRedisSessionSimple("127.0.0.1:6379", 0, "hunterhug")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	userId := "000001"
	token, err := tokenManage.SetToken(userId, 20)
	if err != nil {
		fmt.Println("set token err:", err.Error())
		return
	}
	fmt.Println("set token:", userId, token)

	user, exist, err := tokenManage.CheckToken(token)
	if err != nil {
		fmt.Println("check token err:", err.Error())
		return
	}

	if exist {
		fmt.Printf("check token exist: %#v\n", user)
	} else {
		fmt.Println("check token not exist")
	}

	err = tokenManage.DeleteToken(token)
	if err != nil {
		fmt.Println("delete token err:", err.Error())
		return
	} else {
		fmt.Println("delete token:", token)
	}

	user, exist, err = tokenManage.CheckToken(token)
	if err != nil {
		fmt.Println("after delete check token err:", err.Error())
		return
	}

	if exist {
		fmt.Printf("after delete check delete token exist: %#v\n", user)
	} else {
		fmt.Println("after delete check delete token not exist")
	}

	tokenManage.SetToken(userId, 20)
	tokenManage.SetToken(userId, 20)
	tokenManage.SetToken(userId, 20)
	tokenManage.SetToken(userId, 20)

	tokenList, err := tokenManage.ListUserToken(userId)
	if err != nil {
		fmt.Println("list token err:", err.Error())
		return
	}

	for _, v := range tokenList {
		fmt.Println("list token:", v)
	}

	err = tokenManage.DeleteUserToken(userId)
	if err != nil {
		fmt.Println("delete user all token err:", err.Error())
		return
	} else {
		fmt.Println("delete user all token")
	}

	tokenList, err = tokenManage.ListUserToken(userId)
	if err != nil {
		fmt.Println("after delete user all list token err:", err.Error())
		return
	}

	if len(tokenList) == 0 {
		fmt.Println("user token empty")
	}
}
