package main

import (
	"manago"

	"github.com/jinzhu/gorm"
)

type User struct {
	gorm.Model

	Name     string
	Nickname string

	Groups  []string
	Objects []*Object
}

func (us User) SetRoutes(man *manago.Manager) {

}
func (us User) MigrateDbModel() interface{} {
	return us
}
