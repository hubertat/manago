package main

import (
	"manago"

	"github.com/jinzhu/gorm"
)

type Object struct {
	gorm.Model

	Name        string
	Description string
	Type        int
	Quantity    uint

	Owner *User
}

func (ob Object) SetRoutes(man *manago.Manager) {
	inTechGroup := NewInGroupMiddleware("dzial_tech", "wp_admin")

	man.GET("/object/list", ob.List, &inTechGroup)
	man.GET("/object/show", ob.List)
	man.GET("/", ob.Home)
}

func (ob Object) MigrateDbModel() interface{} {
	return ob
}

func (ob *Object) List(ctr *manago.Controller) error {
	ctr.SetCt("Testing", "123")
	return nil
}

func (ob *Object) Home(ctr *manago.Controller) (err error) {
	ctr.SetCt("Message", "Hello home view")

	return
}
