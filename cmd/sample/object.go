package main

import (
	"manago"
	"manago/cmd/sample/middleware"

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
	inTechGroup := middleware.NewInGroupMiddleware("dzial_tech", "wp_admin")

	man.GET("/object/list", ob.List, &inTechGroup)
	man.AddRoute("GET", "/object/list", ob.List, &inTechGroup)
}

func (ob *Object) SetMiddleware(man *manago.Manager) {

}

func (ob *Object) List(ctr *manago.Controller) error {
	return nil
}
