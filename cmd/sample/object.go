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

func (ob *Object) SetRoutes(man *manago.Manager) {
	man.AddRoute("GET", "/object/list", ob.List)
	man.AddRoute(ob.List, "GET", "/object/list", "object/list.html")
}

func (ob *Object) SetMiddleware(man *manago.Manager) {

}

func (ob *Object) List(ctr *manago.Controller) error {
	return nil
}
