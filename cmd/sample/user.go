package main

import "github.com/jinzhu/gorm"

type User struct {
	gorm.Model

	Name     string
	Nickname string

	Objects []*Object
}
