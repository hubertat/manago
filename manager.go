package manago

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/astaxie/beego/session"
	"github.com/julienschmidt/httprouter"
)

type Manager struct {
	sessionManager *session.Manager
	router         *httprouter.Router

	toManage []Manageable

	Config     Config
	Dbc        *Db
	Clients    map[string]Client
	StaticFsys fs.FS
	Messaging  Messenger
	CronTasks  []Cron

	AppVersion string
	AppBuild   string
}

func New(conf Config, toManage []Manageable, build ...string) (man *Manager, err error) {
	man = &Manager{
		Config:     conf,
		router:     httprouter.New(),
		Dbc:        &Db{},
		Clients:    conf.Clients,
		StaticFsys: os.DirFS("./static"),
	}

	man.toManage = toManage

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

	sessConfig := &session.ManagerConfig{}
	sessConfig.EnableSetCookie = true
	sessConfig.CookieName = "gosessionid"
	sessConfig.Gclifetime = 10000

	man.sessionManager, err = session.NewManager("memory", sessConfig)
	if err != nil {
		err = fmt.Errorf("ERROR Manager New: session config failed: %w", err)
		return
	}

	man.MakeRoutes()

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

	man.router = httprouter.New()
	man.MakeRoutes()

	return
}

func (man *Manager) Migrate() error {
	return man.Dbc.AutoMigrate(man.toManage)
}

func (man *Manager) Start() (status string) {

	go man.sessionManager.GC()

	status = fmt.Sprintf("Manager Start http server: %s:%d\n", man.Config.Server.Host, man.Config.Server.Port)
	go func() {
		log.Print(http.ListenAndServe(fmt.Sprintf("%s:%d", man.Config.Server.Host, man.Config.Server.Port), man.router))
	}()

	if len(man.CronTasks) > 0 {
		status += "Starting Cron loop."
		go man.CronLoop()
	}

	return
}

func (man *Manager) MakeRoutes() {

	man.makeStaticRoutes()

	for _, typ := range man.toManage {
		log.Printf("Manager MakeRoutes: preparing routes for %v", typ)
		typ.SetRoutes(man)
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

type ManagoHandlerFunc func(ctr *Controller) (err error)

type Manageable interface {
	SetRoutes(*Manager)
	MigrateDbModel() interface{}
}

func (man *Manager) GET(path string, handle ManagoHandlerFunc, middlewares ...Middleware) {
	man.AddRoute("GET", path, handle, middlewares...)
}

func (man *Manager) POST(path string, handle ManagoHandlerFunc, middlewares ...Middleware) {
	man.AddRoute("POST", path, handle, middlewares...)
}

func (man *Manager) AddRoute(method string, path string, handle ManagoHandlerFunc, middlewares ...Middleware) {
	man.router.Handle(method, path, man.handleWrapper(handle, path, middlewares...))
}

func (man *Manager) handleWrapper(handlerFunc ManagoHandlerFunc, path string, middlewares ...Middleware) (handle httprouter.Handle) {

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctr := Controller{}

		ctr.Req.SetData(r, ps)

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

		middlewarePermission := true

		if !man.Config.DevSkipMiddleware || man.AppVersion != "v_dev" {
			for _, midWare := range middlewares {
				middlewarePermission = middlewarePermission && midWare.RunBefore(&ctr)
			}
		}

		var errFromHandler error
		if middlewarePermission {
			errFromHandler = handlerFunc(&ctr)
		}

		for _, midWare := range middlewares {
			midWare.RunAfter(&ctr)
		}

		if errFromHandler != nil {
			errCode := ctr.GetError().Code
			log.Printf("handler returned error with code(%d): \n%v", errCode, errFromHandler)
			if errCode == 0 {
				errCode = 500
			}
			http.Error(w, errFromHandler.Error(), errCode)
		} else {
			redirS, redirAddrS := ctr.GetRedir()
			if redirS {
				http.Redirect(w, r, redirAddrS, 303)
			} else {
				errFromTemplate := man.FireTemplate(w, ctr.Ctnt(), path)
				if errFromTemplate != nil {
					log.Println(errFromTemplate.Error())
				}
			}
		}
	}
}
