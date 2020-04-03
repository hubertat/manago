package manago

import (
	"fmt"
	"log"
	"github.com/astaxie/beego/session"
	"github.com/iancoleman/strcase"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"reflect"
	"strings"
)

type Manager struct {
	sessionManager 	*session.Manager
	router         	*httprouter.Router
	

	controllersReflected map[string]reflect.Type
	modelsReflected      map[string]reflect.Type

	Config 	Config
	Views  	*ViewSet
	Dbc    	*Db
	Mid		*MiddlewareManager

	AppVersion		string
	AppBuild		string
}

func New(conf Config, allCtrs []interface{}, allModels []interface{}, build ...string) (man *Manager, err error) {
	man = &Manager{
		Config: conf,
		router: httprouter.New(),
		Dbc:    &Db{},
		Views:  &ViewSet{},
	}

	if len(build) > 0 {
		if len(build[0]) == 0 {
			man.AppVersion = "v_dev"
		} else {
			man.AppVersion = build[0]	
		}
	}
	if len(build) > 1 {
		man.AppBuild = build[1]
	}

	man.controllersReflected = make(map[string]reflect.Type)

	for _, ctr := range allCtrs {
		typ := reflect.ValueOf(ctr).Type()
		if typ.Kind().String() == "ptr" {
			err = fmt.Errorf("ERROR Manager New: Found pointer in allControllers[], expecting struct.")
			return
		}
		name := strcase.ToSnake(typ.Name())
		man.controllersReflected[name] = typ
	}
	log.Print(man.controllersReflected)

	man.modelsReflected = make(map[string]reflect.Type)

	for _, m := range allModels {
		typ := reflect.ValueOf(m).Type()
		if typ.Kind().String() == "ptr" {
			err = fmt.Errorf("ERROR Manager New: Found pointer in allModels[], expecting struct.")
			return
		}
		name := strcase.ToSnake(typ.Name())
		man.modelsReflected[name] = typ
	}

	sessConfig := &session.ManagerConfig{}
	sessConfig.EnableSetCookie = true
	sessConfig.CookieName = "gosessionid"
	sessConfig.Gclifetime = 10000

	man.sessionManager, err = session.NewManager("memory", sessConfig)
	if err != nil {
		err = fmt.Errorf("ERROR Manager New: session config failed: %w", err)
		return
	}

	man.Mid = NewMiddlewareManager()
	man.MakeRoutes()
	man.PrepareMiddlewares()

	err = man.Views.Load(&conf)
	if err != nil {
		err = fmt.Errorf("ERROR Manager New: views set failed: %w", err)
		return
	}


	err = man.Dbc.Check(man.Config.Db)
	if err != nil {
		err = fmt.Errorf("ERROR Manager New: Database check error:\n%v", err)
	}

	return
}

func (man *Manager) Migrate() error {
	return man.Dbc.AutoMigrate(man.modelsReflected)
}

func (man *Manager) Start() (status string) {


	go man.sessionManager.GC()

	status = fmt.Sprintf("Manager Start http server: %s:%d", man.Config.Server.Host, man.Config.Server.Port)
	go func(){
		log.Print(http.ListenAndServe(fmt.Sprintf("%s:%d", man.Config.Server.Host, man.Config.Server.Port), man.router))
	}()

	return
}

func (man *Manager) MakeRoutes() {

	for _, typ := range man.controllersReflected {
		log.Printf("Manager MakeRoutes: preparing routes for %s", typ.Name())
		ctr := reflect.New(typ).Interface().(Controlled)
		ctr.SetManager(man)
		ctr.SetRouter(man.router)
		ctr.SetRoutes()
	}
}

func (man *Manager) PrepareMiddlewares() {

	for _, typ := range man.controllersReflected {
		log.Printf("Manager PrepareMiddleware: preparing middleware for %s", typ.Name())
		ctr := reflect.New(typ).Interface().(Controlled)
		ctr.SetManager(man)
		ctr.PrepareMiddlewares()
	}
}

func (man *Manager) HandleJson(ctrName, mtdName string) httprouter.Handle {

	log.Printf("Manager HandleJson: preparing: %s->%s", ctrName, mtdName)

	typ, isOk := man.controllersReflected[ctrName]
	if !isOk {
		log.Fatalf("Manager HandleJson: controller [%s] not found!", ctrName)
	}

	if !reflect.New(typ).MethodByName(mtdName).IsValid() {
		log.Fatalf("Manager HandleJson: method[%v] not found!", mtdName)
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		ctr := reflect.New(typ).Interface().(Controlled)
		method := reflect.ValueOf(ctr).MethodByName(mtdName)

		log.Printf("%T HandleJson: method[%v]", ctr, mtdName)
	
		ctr.SetReqData(r, ps)
		ctr.SetManager(man)

		dbh, err := ctr.SetupDB(man.Dbc)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dbh.Close()

		err = ctr.StartSession(man.sessionManager, w, r)
		defer ctr.SessionRelease(w)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		method.Call([]reflect.Value{})

		json, errJson := ctr.JsonCtnt()
		if errJson != nil {
			ctr.SetError(500, fmt.Errorf("HandleJson parsing to json failed:\n%v", errJson))
		}

		if ctr.IsError() {
			log.Print("Error from controller detected, serving:")
			log.Print(ctr.GetError().Msg)
			log.Print(ctr.GetError().Err)
			http.Error(w, ctr.GetError().Msg, ctr.GetError().Code)
		} else {
			w.Header().Set("Content-Type", "application/json")
  			w.Write(json)		
		}

	}
}

func (man *Manager) Handle(params ...string) httprouter.Handle {
	var redir bool
	var ctrName, mtdName, tmplName, redirAddr string

	switch len(params) {
	case 4:
		ctrName = params[0]
		mtdName = params[1]
		tmplName = params[2]
		redirAddr = params[3]
		redir = true

	case 2:
		ctrName = params[0]
		mtdName = params[1]
		tmplName = ctrName + "/" + strcase.ToSnake(mtdName)
		redir = false

	case 3:
		ctrName = params[0]
		mtdName = params[1]
		tmplName = params[2]
		redir = false

		if strings.HasPrefix(tmplName, "./") {
			tmplName = strings.Replace(tmplName, ".", ctrName, 1)
		}

	default:
		log.Fatal("Manager Handle: wrong parameter count!")

	}

	log.Printf("Manager Handle: preparing: %v %v ", mtdName, tmplName)

	typ, isOk := man.controllersReflected[ctrName]
	if !isOk {
		log.Fatalf("Manager Handle: controller [%s] not found!", ctrName)
	}

	if !reflect.New(typ).MethodByName(mtdName).IsValid() {
		log.Fatalf("Manager Handle: method[%v] not found! (template: %v)", mtdName, tmplName)
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		ctr := reflect.New(typ).Interface().(Controlled)

		method := reflect.ValueOf(ctr).MethodByName(mtdName)

		log.Printf("%T Handle: method[%v], template[%v]", ctr, mtdName, tmplName)

		ctr.SetReqData(r, ps)

		ctr.SetManager(man)

		dbh, err := ctr.SetupDB(man.Dbc)
		
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dbh.Close()

		err = ctr.StartSession(man.sessionManager, w, r)
		defer ctr.SessionRelease(w)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if redir {
			ctr.SetRedir(redirAddr)
		}

		if man.Mid.ctrRunBefore(ctrName, mtdName, ctr) {
			method.Call([]reflect.Value{})	
		}

		if ctr.IsError() {
			log.Print("Error from controller detected, serving:")
			log.Print(ctr.GetError().Msg)
			log.Print(ctr.GetError().Err)
			http.Error(w, ctr.GetError().Msg, ctr.GetError().Code)
		} else {
			redirS, redirAddrS := ctr.GetRedir()
			if redirS {
				http.Redirect(w, r, redirAddrS, 303)
			} else {
				err := man.Views.FireTemplate(tmplName, w, ctr.Ctnt())
				if err != nil {
					log.Print(err.Error())
				}
			}
		}

	}
}
