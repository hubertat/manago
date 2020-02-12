package controllers

import (
	// "fmt"
	"manago"
)

type Sample struct {
	manago.Controller
}

func (ctr *Sample) SetRoutes() {
	ctr.Name = "sample"
	ctr.Router.GET("/", ctr.Handle("DoNothing"))
}

func (ctr *Sample) DoNothing() {
	ctr.Req.SetCt("Test content", "lalalala")
}