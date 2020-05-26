package rdb

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	//_ "github.com/lib/pq"
	"os"
	"time"
	"xorm.io/core"
	"xorm.io/xorm"
	"xorm.io/xorm/log"
)

// support multi db
const (
	PG    = "postgres"
	MYSQL = "mysql"
)

// db config
type MyDbConfig struct {
	DriverName      string `yaml:"driver_name"`
	MaxIdleCons     int    `yaml:"max_idle_cons"`
	MaxOpenCons     int    `yaml:"max_open_cons"`
	DebugToFile     bool   `yaml:"debug_to_file"`
	DebugToFileName string `yaml:"debug_to_file_path"`
	Debug           bool   `yaml:"debug"`
	DbConfig        `yaml:",inline"`
}

type DbConfig struct {
	Name    string `yaml:"name"`
	Host    string `yaml:"host"`
	User    string `yaml:"user"`
	Pass    string `yaml:"pass"`
	Port    string `yaml:"port"`
	Prefix  string `yaml:"prefix"`
	SslMode string `yaml:"ssl_mode"` // sslmode=verify-full require
}

type MyDb struct {
	Config MyDbConfig
	Client *xorm.Engine
}

// not take db
func NewMysqlUrl(c DbConfig) string {
	if c.Port == "" {
		c.Port = "3306"
	}
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4", c.User, c.Pass, c.Host, c.Port, c.Name)
	return dns
}

func NewMysqlUrl2(c DbConfig) string {
	if c.Port == "" {
		c.Port = "3306"
	}
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/", c.User, c.Pass, c.Host, c.Port)
	return dns
}

func NewPqUrl(c DbConfig) string {
	if c.Port == "" {
		c.Port = "5432"
	}
	//if c.Sslmode == "" {
	//	c.Sslmode = "verify-full"
	//}
	dns := fmt.Sprintf("dbname=%s host=%s user=%s password=%s port=%s sslmode=%s", c.Name, c.Host, c.User, c.Pass, c.Port, c.SslMode)
	return dns
}

func NewDb(config MyDbConfig) (*MyDb, error) {
	if config.DriverName == "" {
		config.DriverName = MYSQL
	}

	db := new(MyDb)
	db.Config = config
	dns := ""
	if config.DriverName == MYSQL {
		if config.DbConfig.Name != "" {
			engine, err := xorm.NewEngine(config.DriverName, NewMysqlUrl2(config.DbConfig))
			if err != nil {
				return db, err
			}
			engine.Exec(fmt.Sprintf("create database %s default character set utf8mb4 collate utf8mb4_unicode_ci;", config.DbConfig.Name))
		}

		if config.DriverName == MYSQL {
			dns = NewMysqlUrl(config.DbConfig)
		}
		if config.DriverName == PG {
			dns = NewPqUrl(config.DbConfig)
		}

		engine, err := xorm.NewEngine(config.DriverName, dns)
		if err != nil {
			return db, err
		}

		if config.Debug {
			if config.DebugToFile {
				if config.DebugToFileName == "" {
					config.DebugToFileName = "/tmp/" + config.DriverName + ".log"
				}
				f, err := os.OpenFile(config.DebugToFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
				if err != nil {
					panic(err)
				}
				engine.SetLogger(log.NewSimpleLogger(f))
			}
			engine.ShowSQL(true)
		}

		engine.TZLocation, _ = time.LoadLocation("Asia/Shanghai")

		if config.Prefix != "" {
			tbMapper := core.NewPrefixMapper(core.SnakeMapper{}, config.Prefix)
			engine.SetTableMapper(tbMapper)
		}

		engine.SetMaxIdleConns(config.MaxIdleCons)
		engine.SetMaxOpenConns(config.MaxOpenCons)

		if err := engine.Ping(); err != nil {
			return db, err
		}
		db.Client = engine
		return db, nil
	} else {
		return db, errors.New("Not support this drive:" + config.DriverName)
	}
}

func (db *MyDb) Ping() error {
	if db.Client == nil {
		return errors.New("client nil")
	} else {
		return db.Client.Ping()
	}
}

func (db *MyDb) IsTableExist(beanOrTableName interface{}) (bool, error) {
	return db.Client.IsTableExist(beanOrTableName)

}

func (db *MyDb) DropTables(beans ...interface{}) error {
	err := db.Client.DropTables(beans...)
	return err
}

func (db *MyDb) CreateTables(beanOrTableName interface{}) error {
	err := db.Client.CreateTables(beanOrTableName)
	return err
}

func (db *MyDb) Sync2(beanOrTableName interface{}) error {
	err := db.Client.Sync2(beanOrTableName)
	return err
}

func (db *MyDb) Insert(beans ...interface{}) (int64, error) {
	return db.Client.Insert(beans...)
}

func (db *MyDb) InsertOne(beans interface{}) (int64, error) {
	return db.Client.InsertOne(beans)
}

func (db *MyDb) Update(bean interface{}, condBean ...interface{}) (int64, error) {
	return db.Client.Update(bean, condBean...)

}

func (db *MyDb) Delete(bean interface{}) (int64, error) {
	return db.Client.Delete(bean)

}

// sql := "select * from userinfo"
// results, err := engine.Query(sql)
func (db *MyDb) Query(sql string, paramStr ...interface{}) (resultsSlice []map[string][]byte, err error) {
	return db.Client.Query(sql, paramStr)

}

// sql = "update `userinfo` set username=? where id=?"
// res, err := engine.Exec(sql, "xiaolun", 1)
func (db *MyDb) Exec(sql string, args ...interface{}) (sql.Result, error) {
	return db.Client.Exec(sql, args)
}
