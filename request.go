package manago

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"reflect"
	"strconv"
	"time"
	"net/http"
	"strings"
	"log"
)

type Request struct {
	ViewContent map[string]interface{}
	R           *http.Request
	Id          int

	redir        bool
	redirAddress string
	params       httprouter.Params
}

func (req *Request) SetData(r *http.Request, ps httprouter.Params) {
	
	req.ViewContent = make(map[string]interface{})
	req.R = r
	req.params = ps

	req.R.ParseForm()

	id, err := strconv.ParseUint(ps.ByName("id"), 10, 32)
	if err == nil && id > 0 {
		req.Id = int(id)
	}

}

func (req *Request) SetRedir(addr string) {
	req.redir = true
	req.redirAddress = addr
}

func (req *Request) AppendRedir(id uint) {
	req.redirAddress = fmt.Sprintf("%s%d", req.redirAddress, id)
}

func (req *Request) AppendRedirS(s string) {
	req.redirAddress = req.redirAddress + s
}

func (req *Request) GetRedir() (redir bool, addr string) {
	redir = req.redir
	addr = req.redirAddress

	return
}
func (req *Request) Ctnt() *map[string]interface{} {
	return &req.ViewContent
}

func (req *Request) SetCt(name string, val interface{}) {
	req.ViewContent[name] = val
}

func (req *Request) SetCtQuick(val interface{}, pms ...string) {
	var name string
	if len(pms) == 0 {
		typ := reflect.ValueOf(val).Type()
		if typ.Kind().String() == "ptr" {
			name = reflect.Indirect(reflect.ValueOf(val)).Type().Name()
		} else {
			name = typ.Name()
		}
	} else {
		name = pms[0]
	}
	req.SetCt(name, val)
}

func (req *Request) FormCheckSingle(fields ...string) bool {

	for _, name := range fields {
		if len(req.FormSingle(name)) > 0 {
			return true
		}
	}
	return false
}
func (req *Request) FormSingle(name string) string {
	elem, exist := req.R.Form[name]
	if exist {
		return elem[0]
	}

	return ""
}

func (req *Request) FormSlice(name string) []string {
	elem, exist := req.R.Form[name]
	if exist {
		return elem
	}

	return []string{}
}

func (req *Request) FormFloat(name string) float64 {
	val, err := strconv.ParseFloat(strings.Replace(req.FormSingle(name), ",", ".", 1), 64)
	if err == nil {
		return val
	}

	return 0
}

func (req *Request) FormInt(name string) uint {

	iVal, iErr := strconv.ParseInt(req.FormSingle(name), 10, 32)
	// log.Printf("Trying FormInt for %s, effect %d", name, iVal)
	if iErr == nil {
		return uint(iVal)
	}

	return 0
}

func (req *Request) FormDate(name string, tm *time.Time) bool {
	tParsed, err := time.Parse("2006-01-02", req.FormSingle(name))
	if err == nil {
		hundretYears, _ := time.ParseDuration("876000h")
		if time.Since(tParsed) > hundretYears {
			return false
		}

		*tm = tParsed
		return true
	} else {
		log.Print(err)
		return false
	}
}

func (req *Request) ParamByName(name string) string {
	return req.params.ByName(name)
}

func (req *Request) ParamIntByName(name string) int {
	num, err := strconv.ParseUint(req.ParamByName(name), 10, 64)

	if err == nil {
		return int(num)
	} else {
		log.Print("ParamIntByName: conversion to int error!")
		log.Print(err)
		return 0
	}
}
