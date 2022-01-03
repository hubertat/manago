package manago

import (
	"fmt"
	"log"
	"reflect"
	"time"
)

type Cron struct {
	Minute  *int
	Hour    *int
	Weekday *int

	ControllerName string
	MethodName     string

	lastRun time.Time
}

func NewCron(ctr string, mtd string, when ...int) (task Cron) {

	if len(ctr) > 0 && len(mtd) > 0 {
		task = Cron{ControllerName: ctr, MethodName: mtd}
	} else {
		task = Cron{}
		return
	}

	switch len(when) {
	case 3:
		min := when[0]
		hr := when[1]
		wd := when[2]
		task.Minute = &min
		task.Hour = &hr
		task.Weekday = &wd
	case 2:
		min := when[0]
		hr := when[1]
		task.Minute = &min
		task.Hour = &hr
	case 1:
		min := when[0]
		task.Minute = &min
	}

	return
}

func (cr *Cron) CheckTime() bool {
	now := time.Now()
	if !cr.lastRun.IsZero() {
		if time.Since(cr.lastRun) < time.Minute {
			return false
		}
	}

	if cr.Weekday != nil && *cr.Weekday != int(now.Weekday()) {
		return false
	}

	if cr.Hour != nil && *cr.Hour != now.Hour() {
		return false
	}

	if cr.Minute != nil && *cr.Minute != now.Minute() {
		return false
	}

	return true
}

func (cr *Cron) RunMethod(man *Manager) error {
	cr.lastRun = time.Now()

	typ, isOk := man.controllersReflected[cr.ControllerName]
	if !isOk {
		return fmt.Errorf("Cron checkMethod: controller [%s] not found!", cr.ControllerName)
	}

	if !reflect.New(typ).MethodByName(cr.MethodName).IsValid() {
		return fmt.Errorf("Cron checkMethod: method[%v] not found!", cr.MethodName)
	}

	ctr := reflect.New(typ).Interface().(Controlled)

	method := reflect.ValueOf(ctr).MethodByName(cr.MethodName)
	ctr.SetManager(man)
	ctr.SetEmptyReq()
	_, err := ctr.SetupDB(man.Dbc)

	if err != nil {
		return fmt.Errorf("Cron checkMethod failed to setup db: \n%v\n", err.Error())
	}

	log.Printf("##Cron RunMethod passed, running %s from controller %s.\n", cr.MethodName, cr.ControllerName)
	go method.Call([]reflect.Value{})
	return nil
}
