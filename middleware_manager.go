package manago

import (
	"fmt"
	"log"
	"strings"
)

type Middleware interface {
	RunBefore(Controlled, map[string]string) bool
	RunAfter(Controlled, map[string]string)
}

type MidMethodSet struct {
	metName    string
	middleware Middleware
	params     map[string]string
}

type MidCtrSet struct {
	ctrName string
	methods []MidMethodSet
}

type MiddlewareManager struct {
	ctrMap map[string]*MidCtrSet
}

func NewMiddlewareManager() *MiddlewareManager {
	mm := MiddlewareManager{}
	mm.ctrMap = make(map[string]*MidCtrSet)

	return &mm
}

func (mm *MiddlewareManager) ControllerSetRaw(ctrName string, middleware Middleware, params map[string]string, methods ...string) error {
	ms := MidMethodSet{
		middleware: middleware,
		params:     params,
	}

	return mm.ControllerSet(ctrName, &ms, methods...)
}

// ControllerSet will attach middleware with params to selected methods of calling controller
func (mm *MiddlewareManager) ControllerSet(ctrName string, set *MidMethodSet, methods ...string) error {
	if len(methods) == 0 {
		return fmt.Errorf("MiddlewareManager Add: empty methods input")
	}

	cs, present := mm.ctrMap[ctrName]

	if !present {
		mm.ctrMap[ctrName] = &MidCtrSet{ctrName: ctrName}
		cs, _ = mm.ctrMap[ctrName]
		cs.methods = []MidMethodSet{}
	}

	var ms MidMethodSet

	for _, metName := range methods {
		ms = *set
		ms.metName = metName
		cs.methods = append(cs.methods, ms)
	}

	return nil
}

func (mm *MiddlewareManager) GetSet(middleware Middleware, params ...string) *MidMethodSet {
	ms := &MidMethodSet{
		middleware: middleware,
	}

	ms.params = make(map[string]string)
	for _, pm := range params {
		pSlice := strings.SplitN(pm, ":", 2)
		if len(pSlice) > 1 {
			ms.params[pSlice[0]] = pSlice[1]
		} else {
			ms.params[pSlice[0]] = ""
		}

	}

	return ms
}

func (mm *MiddlewareManager) ctrRunBefore(ctrName, mtdName string, ctr Controlled) (proceed bool) {
	midCtr, ok := mm.ctrMap[ctrName]

	proceed = true

	if ok {
		for _, ms := range midCtr.methods {
			if ms.metName == mtdName || ms.metName == "_all" {
				log.Printf("Manago MiddlewareManager: running %T with %v\n", ms.middleware, ms.params)
				proceed = proceed && ms.middleware.RunBefore(ctr, ms.params)
			}
		}
	}

	if !proceed {
		log.Println("Manago MiddlewareManager: middleware finished, requested method will not proceed")
	}

	return
}

func (mm *MiddlewareManager) ctrRunAfter(ctrName, mtdName string, ctr Controlled) {
	midCtr, ok := mm.ctrMap[ctrName]

	if ok {
		for _, ms := range midCtr.methods {
			if ms.metName == mtdName || ms.metName == "_all" {
				log.Printf("Manago MiddlewareManager: running %T with %v\n", ms.middleware, ms.params)
				ms.middleware.RunAfter(ctr, ms.params)
			}
		}
	}

	return
}
