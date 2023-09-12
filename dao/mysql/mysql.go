package mysql

import (
	"ForumSystem/models"
	"ForumSystem/setting"
	"fmt"
	"github.com/go-xorm/xorm"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const (
	driveName = "mysql"
	dsName    = "root:abc123@(127.0.0.1:3306)/ForumSystem?charset=utf8"
	showSQL   = true
	maxCon    = 10
	NONERROR  = "noerror" //没有错误
)

var db *sqlx.DB
var DbEngin *xorm.Engine

// Init 初始化MySQL连接
func Init(cfg *setting.MySQLConfig) (err error) {
	// "user:password@tcp(host:port)/dbname"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DB)
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		return
	}
	DbEngin, err = xorm.NewEngine(driveName, dsName)
	if nil != err && NONERROR != err.Error() {
		log.Fatal(err.Error())
	}
	//是否显示SQL语句
	DbEngin.ShowSQL(showSQL)
	//自动User
	DbEngin.Sync2(new(models.Contact),
		new(models.ChatCommunity))
	fmt.Println("init data base ok")

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	return
}

// Close 关闭MySQL连接
func Close() {
	_ = db.Close()
}
