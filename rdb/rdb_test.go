package rdb

import (
	"fmt"
	"testing"
)

type TestUser struct {
	Id     int64  `json:"id" xorm:"pk"`
	Detail string `json:"detail"`
}

func TestNewDb(t *testing.T) {
	c := MyDbConfig{
		MaxIdleCons: 15,
		MaxOpenCons: 15,
		Debug:       true,
		DbConfig: DbConfig{
			Name:   "rdb_db",
			Host:   "127.0.0.1",
			User:   "root",
			Pass:   "xxx",
			Port:   "3306",
			Prefix: "xx_",
		},
	}
	client, err := NewDb(c)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = client.CreateTables(new(TestUser))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	_, err = client.InsertOne(TestUser{
		Detail: "",
	})

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	users := make([]TestUser, 0)
	err = client.Client.Find(&users)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, v := range users {
		fmt.Println(v)
	}
}
