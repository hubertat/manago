package manago

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/astaxie/beego/session"
	"github.com/iancoleman/strcase"
	"github.com/julienschmidt/httprouter"
)

type Manager struct {
	sessionManager *session.Manager
	router         *httprouter.Router

	controllersReflected map[string]reflect.Type
	modelsReflected      map[string]reflect.Type

	Config     Config
	Views      *ViewSet
	Dbc        *Db
	Mid        *MiddlewareManager
	Clients    map[string]Client
	StaticFsys fs.FS
	Messaging  Messenger
	CronTasks  []Cron

	AppVersion string
	AppBuild   string
}

func New(conf Config, allCtrs []interface{}, allModels []interface{}, build ...string) (man *Manager, err error) {
	man = &Manager{
		Config:     conf,
		router:     httprouter.New(),
		Dbc:        &Db{},
		Views:      &ViewSet{},
		Clients:    conf.Clients,
		StaticFsys: os.DirFS("./static"),
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

	err = man.Views.Load(&conf, man)
	if err != nil {
		err = fmt.Errorf("ERROR Manager New: views set failed: %w", err)
		return
	}

	err = man.Dbc.Check(man.Config.Db)
	if err != nil {
		err = fmt.Errorf("ERROR Manager New: Database check error:\n%v", err)
	}

	if conf.SlackHook != nil {
		man.Messaging = &Slack{HookUrl: *conf.SlackHook}
	}

	return
}

func (man *Manager) ReloadStaticFS(fsys fs.FS) (err error) {
	log.Println("Reloading Static FS, will reload Views and make new httprouter.")

	if fsys == nil {
		err = fmt.Errorf("Manager Reloading static FS: received nil!")
		return
	}
	subStatic, errSub := fs.Sub(fsys, "static")
	if errSub == nil {
		man.StaticFsys = subStatic
	} else {
		man.StaticFsys = fsys
	}

	err = man.Views.Load(&man.Config, man)
	if err != nil {
		err = fmt.Errorf("Manager Reloading static FS: views set failed: %w", err)
		return
	}

	man.router = httprouter.New()
	man.MakeRoutes()

	return
}

func (man *Manager) Migrate() error {
	return man.Dbc.AutoMigrate(man.modelsReflected)
}

func (man *Manager) Start() (status string) {

	go man.sessionManager.GC()

	status = fmt.Sprintf("Manager Start http server: %s:%d\n", man.Config.Server.Host, man.Config.Server.Port)
	if len(man.Config.Server.RedirectFromPorts) > 0 {
		status = fmt.Sprintf("%s + redirecting from ports: %v\n", status, man.Config.Server.RedirectFromPorts)
	}

	for _, redirPort := range man.Config.Server.RedirectFromPorts {
		redirS := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", man.Config.Server.Host, redirPort),
			Handler: http.RedirectHandler(fmt.Sprintf("http://%s:%d", man.Config.Server.Host, man.Config.Server.Port), 301),
		}
		go func() {
			log.Print(redirS.ListenAndServe())
		}()
	}
	go func() {
		log.Print(http.ListenAndServe(fmt.Sprintf("%s:%d", man.Config.Server.Host, man.Config.Server.Port), man.router))
	}()

	if len(man.CronTasks) > 0 {
		status += "Starting Cron loop."
		go man.CronLoop()
	}

	return
}

func (man *Manager) StartTls(certFile string, keyFile string) (status string) {

	go man.sessionManager.GC()

	status = fmt.Sprintf("Manager Start http server: %s:%d, with cert file: %s and key file: %s\n", man.Config.Server.Host, man.Config.Server.Port, certFile, keyFile)
	go func() {
		log.Print(http.ListenAndServeTLS(fmt.Sprintf("%s:%d", man.Config.Server.Host, man.Config.Server.Port), certFile, keyFile, man.router))
	}()

	if len(man.CronTasks) > 0 {
		status += "Starting Cron loop."
		go man.CronLoop()
	}

	return
}

func (man *Manager) MakeRoutes() {

	man.makeStaticRoutes()

	for _, typ := range man.controllersReflected {
		log.Printf("Manager MakeRoutes: preparing routes for %s", typ.Name())
		ctr := reflect.New(typ).Interface().(Controlled)
		ctr.SetManager(man)
		ctr.SetRouter(man.router)
		ctr.SetRoutes()
	}
}

func (man *Manager) makeStaticRoutes() error {

	if man.StaticFsys == nil {
		return fmt.Errorf("makeStaticRoutes have nil static FS, stop")
	}

	staticFiles, statErr := fs.Sub(man.StaticFsys, man.Config.WebStaticPath)
	if statErr != nil {
		return statErr
	}

	log.Printf("Manager MakeRoutes: found static files to serve, setting up. \n")
	man.router.ServeFiles("/static/*filepath", http.FS(staticFiles))

	return nil
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

		var middlewarePermission bool

		if man.Config.DevSkipMiddleware && man.AppVersion == "v_dev" {
			middlewarePermission = true
		} else {
			middlewarePermission = man.Mid.ctrRunBefore(ctrName, mtdName, ctr)
		}

		if middlewarePermission {
			method.Call([]reflect.Value{})
		}

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
		requestStarted := time.Now()
		ctr := reflect.New(typ).Interface().(Controlled)

		method := reflect.ValueOf(ctr).MethodByName(mtdName)

		log.Printf("%T Handle: method[%v], template[%v]", ctr, mtdName, tmplName)

		ctr.SetReqData(r, ps)

		ctr.SetManager(man)

		ctr.SetRequestStartTime(&requestStarted)

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

		var middlewarePermission bool

		if man.Config.DevSkipMiddleware && man.AppVersion == "v_dev" {
			middlewarePermission = true
		} else {
			middlewarePermission = man.Mid.ctrRunBefore(ctrName, mtdName, ctr)
		}

		if middlewarePermission {
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
				http.Redirect(w, r, redirAddrS, http.StatusSeeOther)
			} else {
				ctr.FillExecutionTime()
				err := man.Views.FireTemplate(tmplName, w, ctr.Ctnt())
				if err != nil {
					log.Print(err.Error())
				}
			}
		}

	}
}

func (man *Manager) HandleDirect(params ...string) httprouter.Handle {

	var ctrName, mtdName string

	switch len(params) {
	case 2:
		ctrName = params[0]
		mtdName = params[1]

	default:
		log.Fatal("Manager HandleDirect: wrong parameter count!")

	}

	log.Printf("Manager HandleDirect: preparing: %v", mtdName)

	typ, isOk := man.controllersReflected[ctrName]
	if !isOk {
		log.Fatalf("Manager HandleDirect: controller [%s] not found!", ctrName)
	}

	if !reflect.New(typ).MethodByName(mtdName).IsValid() {
		log.Fatalf("Manager HandleDirect: method[%v] not found!", mtdName)
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		ctr := reflect.New(typ).Interface().(Controlled)

		method := reflect.ValueOf(ctr).MethodByName(mtdName)

		input := []reflect.Value{
			reflect.ValueOf(w),
			reflect.ValueOf(r),
			reflect.ValueOf(ps),
		}

		log.Printf("%T HandleDirect: method[%v]", ctr, mtdName)

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

		var middlewarePermission bool

		if man.Config.DevSkipMiddleware && man.AppVersion == "v_dev" {
			middlewarePermission = true
		} else {
			middlewarePermission = man.Mid.ctrRunBefore(ctrName, mtdName, ctr)
		}

		if middlewarePermission {
			method.Call(input)
		}

		if ctr.IsError() {
			log.Print("Error from controller detected, serving:")
			log.Print(ctr.GetError().Msg)
			log.Print(ctr.GetError().Err)
			http.Error(w, ctr.GetError().Msg, ctr.GetError().Code)
		}

	}
}

func FileDirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return true, err
}

func (man *Manager) CronLoop() {
	for {
		for taskIndex, _ := range man.CronTasks {
			if man.CronTasks[taskIndex].CheckTime() {
				err := man.CronTasks[taskIndex].RunMethod(man)
				if err != nil {
					log.Printf("Error from Cron Task: %v\n", err)
				}
			}
		}
		time.Sleep(5 * time.Second)
	}

}

func (man *Manager) CopyDatabase() {
	log.Println("Copy Database")
	log.Println("will perform migration on target db and then copy all rows from source (main) db")

	if man.Config.DbTarget == nil {
		log.Println("Target db not configured, cannot continue.")
		return
	}

	targetDb := &Db{}
	err := targetDb.Check(*man.Config.DbTarget)
	if err != nil {
		log.Println("Failed to load target db config, cannot continue:")
		log.Println(err)
		return
	}

	log.Println("migrating target db...")
	start := time.Now()
	err = targetDb.AutoMigrate(man.modelsReflected)
	if err != nil {
		log.Println("Failed to perform migration for target db, will not copy db:")
		log.Println(err)
		return
	}
	log.Println("finished migration in ", time.Since(start).Seconds(), " seconds.")

	log.Println("copying rows...")
	start = time.Now()

	err = man.Dbc.CopyDb(man.modelsReflected, targetDb)
	if err != nil {
		log.Println("Received an error during copying db to target, most likely copy is not complete:")
		log.Println(err)
	} else {
		log.Println("Copy db to target complete (took ", time.Since(start).Seconds(), " seconds).")
	}

}
