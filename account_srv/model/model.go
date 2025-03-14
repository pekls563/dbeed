package model

import "gorm.io/gorm"

type Account struct {
	gorm.Model
	//Mobile   string `gorm:"index:idx_mobile;unique;varchar(11);not null"`

	//PasswordMd string `gorm:"type:varchar("`
	UserName string `gorm:"type:varchar(32)"`
	NickName string `gorm:"type:varchar(32)"`
	Password string `gorm:"type:varchar(128)"`
	Salt     string `gorm:"type:varchar(32)"`

	//Salt     string `gorm:"type:varchar(16)"`
	//Gender   string `gorm:"varchar(6);default:male"`
	//Role     int    `gorm:"type:int;default:1;comment'1-普通用户,2-管理员'"`
	Email     string `gorm:"type:varchar(32)"`
	UserPic   string `gorm:"type:varchar(32)"`
	AvatarUrl string `gorm:"type:varchar(128)"`
}
