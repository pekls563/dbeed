package internal

import (
	"bigEventProject/account_srv/model"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"log"
	"os"

	"time"
)

var DB *gorm.DB
var err error

//func init() {
//	development, _ := zap.NewDevelopment()
//	zap.ReplaceGlobals(development)
//
//}

func InitDB() {

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	//dsn := "root:123456@tcp(localhost:3306)/itheima?charset=utf8mb4&parseTime=True&loc=Local"
	conn := "root:123456@tcp(localhost:3306)/da_shi_jian_account?charset=utf8mb4&parseTime=True&loc=Local"
	//conn := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", myconfig.AppConf.AccountDB.UserName, myconfig.AppConf.AccountDB.Password, myconfig.AppConf.AccountDB.Host, myconfig.AppConf.AccountDB.Port, myconfig.AppConf.AccountDB.DBName)
	//zap.S().Infof(conn)
	DB, err = gorm.Open(mysql.Open(conn), &gorm.Config{
		Logger: newLogger,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, //表明用英文单数形式
		},
	})
	if err != nil {
		panic("数据库连接失败:" + err.Error())
	}
	fmt.Println("连接成功....")
	err = DB.AutoMigrate(&model.Account{})
	if err != nil {
		panic("Account建表失败")
	}

}
