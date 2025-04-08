package model

import "gorm.io/gorm"

type Category struct {
	gorm.Model
	CategoryName  string `gorm:"varchar(32)"`
	CategoryAlias string `gorm:"varchar(32)"`
}

type Article struct {
	gorm.Model
	Title      string `gorm:"varchar(32)"`
	Content    string `gorm:"varchar(32)"`
	CoverImg   string `gorm:"varchar(32)"`
	State      string `gorm:"varchar(32)"`
	CategoryId int    `gorm:"type:int"`
}
