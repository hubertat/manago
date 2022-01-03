package manago

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/astaxie/beego/session"
	"github.com/iancoleman/strcase"
	"github.com/julienschmidt/httprouter"
	"gorm.io/gorm"
)

type Controlled interface {
	SetRoutes()
	PrepareMiddlewares()
	GetMiddleware(Middleware, ...string) *MidMethodSet
	SetMiddleware(*MidMethodSet, ...string) error
	SetMiddlewareDirect(Middleware, ...string) error
	SetMiddlewareParams(Middleware, map[string]string, ...string) error
	Handle(...string) httprouter.Handle
	SetupDB(*Db) (*gorm.DB, error)
	StartSession(*session.Manager, http.ResponseWriter, *http.Request) error
	SessionRelease(http.ResponseWriter)
	IsError() bool
	GetError() StatusError
	SetError(int, error, ...string)
	ClearError()
	SetManager(*Manager)
	SetRouter(*httprouter.Router)
	SetReqData(*http.Request, httprouter.Params)
	SetEmptyReq()
	Ctnt() *map[string]interface{}
	JsonCtnt() ([]byte, error)
	GetRedir() (bool, string)
	SetRedir(string)
	AuthGetUser(interface{}, ...string) error
	FirstPreload(interface{}, uint, ...string) error
	GetModel(interface{}, ...string) error
	GetAltDbConfig() *DatabaseConfig
	CallClient(string, string, url.Values, interface{}) error
	VerifyApiKey() bool
	QuickSendMessage(string) error
	SetRequestStartTime(*time.Time)
	FillExecutionTime()
}

type File interface {
	IsTemporary() bool
	MoveTemp(...string) error
	Reset()
}

type Auth struct {
	IsIn     bool
	Username string
	Guid     string
}

type StatusError struct {
	Code int
	Err  error
	Msg  string
}

type Controller struct {
	// controller definition data
	Name        string
	modelObject interface{}

	Router *httprouter.Router

	// session specific data
	Session session.Store
	Auth    Auth
	Req     Request
	Db      *gorm.DB
	E       StatusError

	Man *Manager
}

func (ctr *Controller) SetRoutes()          {}
func (ctr *Controller) PrepareMiddlewares() {}
func (ctr *Controller) GetPaginator() *Paginator {
	return NewPaginator(ctr)
}

func (ctr *Controller) SetReqData(r *http.Request, ps httprouter.Params) {
	ctr.Req.SetData(r, ps)
}

func (ctr *Controller) SetEmptyReq() {
	ctr.Req.SetData(nil, httprouter.Params{})
}

func (ctr *Controller) Ctnt() *map[string]interface{} {
	return ctr.Req.Ctnt()
}

func (ctr *Controller) JsonCtnt() ([]byte, error) {
	return json.Marshal(ctr.Ctnt())
}

func (ctr *Controller) SetCt(name string, val interface{}) {
	ctr.Req.SetCt(name, val)
}

func (ctr *Controller) GetRedir() (bool, string) {
	return ctr.Req.GetRedir()
}

func (ctr *Controller) SetRedir(input string) {
	ctr.Req.SetRedir(input)
}

func (ctr *Controller) HandleJson(mtdName string) httprouter.Handle {
	return ctr.Man.HandleJson(ctr.Name, mtdName)
}

func (ctr *Controller) HandleDirect(mtdName string) httprouter.Handle {
	return ctr.Man.HandleDirect(ctr.Name, mtdName)
}

func (ctr *Controller) Handle(options ...string) httprouter.Handle {

	options = append([]string{ctr.Name}, options...)
	return ctr.Man.Handle(options...)
}

func (ctr *Controller) GetMiddleware(middleware Middleware, params ...string) *MidMethodSet {
	return ctr.Man.Mid.GetSet(middleware, params...)
}
func (ctr *Controller) SetMiddleware(mms *MidMethodSet, methods ...string) error {
	return ctr.Man.Mid.ControllerSet(ctr.Name, mms, methods...)
}
func (ctr *Controller) SetMiddlewareParams(mid Middleware, params map[string]string, methods ...string) error {
	return ctr.Man.Mid.ControllerSetRaw(ctr.Name, mid, params, methods...)
}

func (ctr *Controller) SetMiddlewareDirect(mid Middleware, methods ...string) error {
	params := make(map[string]string)
	return ctr.Man.Mid.ControllerSetRaw(ctr.Name, mid, params, methods...)
}

// func (ctr *Controller) File(options  ...string) httprouter.Handle {
// 	options = append([]string{ctr.Name}, options...)
// 	return Fire(options...)
// }

func (ctr *Controller) SetManager(man *Manager) {
	ctr.Man = man
}

func (ctr *Controller) SetupDB(dbc *Db) (*gorm.DB, error) {
	var err error
	ctr.Db, err = dbc.Open()
	if err != nil {
		return nil, err
	}
	ctr.Db = ctr.Db.Set("gorm:auto_preload", false)

	return ctr.Db, err
}

func (ctr *Controller) SetRouter(r *httprouter.Router) {
	ctr.Router = r
}

func (ctr *Controller) StartSession(s *session.Manager, w http.ResponseWriter, r *http.Request) error {
	var err error
	ctr.Session, err = s.SessionStart(w, r)
	if err != nil {
		return err
	}
	auth := ctr.Session.Get("auth")
	switch auth := auth.(type) {
	default:
		ctr.Auth = Auth{}

	case string:
		ctr.Auth.IsIn = true
		ctr.Auth.Guid = auth

	}

	username := ctr.Session.Get("username")
	switch username := username.(type) {
	case string:
		ctr.Auth.Username = username
	}

	ctr.Req.SetCtQuick(ctr.Auth)
	ctr.Req.SetCt("AppVersion", ctr.Man.AppVersion)
	ctr.Req.SetCt("AppBuild", ctr.Man.AppBuild)
	return nil
}

func (ctr *Controller) SessionRelease(w http.ResponseWriter) {
	ctr.Session.SessionRelease(w)
}

func (ctr *Controller) IsError() bool {
	if ctr.E.Code > 399 {
		return true
	}

	return false
}

func (ctr *Controller) GetError() StatusError {
	return ctr.E
}

func (ctr *Controller) SetError(code int, err error, msg ...string) {
	ctr.E.Code = code
	ctr.E.Err = err
	if len(msg) > 0 {
		ctr.E.Msg = msg[0]
		if err == nil {
			ctr.E.Err = errors.New(msg[0])
		}
	} else {
		if err != nil {
			ctr.E.Msg = err.Error()
		}
	}

	log.Printf("BaseController SetError fired, received:\n= %d\n= %v\n= %v\n", code, err, msg)
}

func (ctr *Controller) ClearError() {
	ctr.E.Code = 0
	ctr.E.Err = nil
	ctr.E.Msg = ""
}

func (ctr *Controller) FillModel(model interface{}) int {

	valuesCount := 0
	v := reflect.ValueOf(model).Elem()

	for ix := 0; ix < v.NumField(); ix++ {
		fieldName := v.Type().Field(ix).Name
		vals, exist := ctr.Req.R.Form[strcase.ToSnake(fieldName)]

		if v.Field(ix).CanSet() && exist && len(vals) > 0 && fieldName != "ID" {
			formVal := vals[0]
			switch v.Field(ix).Interface().(type) {

			case string:
				if len(formVal) > 0 {
					v.Field(ix).SetString(formVal)
					valuesCount++
				}

			case int:
				iVal, iErr := strconv.ParseInt(formVal, 10, 64)
				if iVal > 0 && iErr == nil {
					v.Field(ix).SetInt(iVal)
					valuesCount++
				}
			case uint:
				iVal, iErr := strconv.ParseUint(formVal, 10, 64)
				if iVal > 0 && iErr == nil {
					v.Field(ix).SetUint(iVal)
					valuesCount++
				}
			case bool:
				bVal, err := strconv.ParseBool(formVal)
				if err == nil {
					v.Field(ix).SetBool(bVal)
					valuesCount++
				} else {
					if formVal == "on" {
						v.Field(ix).SetBool(true)
						valuesCount++
					}
				}

			case time.Time:
				time, err := time.Parse("2006-01-02", formVal)
				if err == nil {
					v.Field(ix).Set(reflect.ValueOf(time))
					valuesCount++
				}
			default:

			}
		}
	}

	log.Printf("FillModel for %T, values filled: %d", model, valuesCount)

	return valuesCount
}

func (ctr *Controller) GetModel(model interface{}, preload ...string) (err error) {
	var modelId uint

	kind := reflect.ValueOf(model).Type().Kind().String()
	if kind != "ptr" {
		return errors.New("Expected pointer input! Received non-pointer type.")
	}
	modelName := reflect.Indirect(reflect.ValueOf(model)).Type().Name()
	snakeName := strcase.ToSnake(modelName)

	// log.Printf("Model name: %s, Snake Name: %s", modelName, snakeName)

	formFieldName := snakeName + "_id"

	if ctr.Req.Id > 0 {
		modelId = uint(ctr.Req.Id)
		// log.Printf("Non zero Data.Id: %d", modelId)
	} else {
		if ctr.Req.FormInt(snakeName+"_id") > 0 {
			modelId = ctr.Req.FormInt(formFieldName)
			// log.Printf("Non zero form snakeName id: %d", modelId)
		}
	}

	if modelId == 0 {
		err = fmt.Errorf("GetModel failed, id not found (Req.Id: %d, form: %s = %d)", ctr.Req.Id, formFieldName, ctr.Req.FormInt(formFieldName))
		ctr.SetError(400, err)
		return
	}

	tx := ctr.Db

	for _, field := range preload {
		tx = tx.Preload(field)
	}

	err = tx.First(model, modelId).Error
	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctr.SetError(404, err)
			return
		} else {
			ctr.SetError(500, err)
		}
	}
	return
}

func (ctr *Controller) FirstPreload(model interface{}, modelId uint, preload ...string) (err error) {
	tx := ctr.Db

	for _, pre := range preload {
		tx = tx.Preload(pre)
	}

	preloadErr := tx.First(model, modelId).Error
	if preloadErr != nil {
		err = fmt.Errorf("BaseController FirstPreload (|%d) failed: \n%v", modelId, preloadErr)
	}

	return
}

func (ctr *Controller) LookForFileponds(model interface{}, file File, params ...string) (fParsed int, err error) {
	type FileId struct {
		Id uint
	}

	kindF := reflect.ValueOf(file).Type().Kind().String()
	if kindF != "ptr" {
		err = fmt.Errorf("Expected file as pointer! Received non-pointer type.")
		return
	}
	// fType := reflect.Indirect(reflect.ValueOf(file)).Type()
	kind := reflect.ValueOf(model).Type().Kind().String()
	if kind != "ptr" {
		err = fmt.Errorf("Expected pointer model input! Received non-pointer type.")
		return
	}
	modelName := reflect.Indirect(reflect.ValueOf(model)).Type().Name()
	var pathName string
	var nestedPath bool
	if len(params) == 0 {
		pathName = strcase.ToSnake(modelName)
	} else {
		pathName = params[0]
		nestedPath = true
	}

	var fId *FileId
	var cnt int64

	for _, onePond := range ctr.Req.FormSlice("filepond") {
		if len(onePond) > 0 {
			fId = &FileId{}
			cnt = 0
			err = json.Unmarshal([]byte(onePond), fId)
			if err != nil {
				err = fmt.Errorf("BaseController LookForFileponds: decoding file id error: %w", err)
				return
			}
			// file := fileIn
			file.Reset()
			ctr.Db.First(file, fId.Id).Count(&cnt)
			if cnt == 0 {
				err = fmt.Errorf("BaseController LookForFileponds: file (%d) not found", fId.Id)
				return
			}
			if !file.IsTemporary() {
				err = fmt.Errorf("BaseController LookForFileponds: file (%d) is not TempFile!", fId.Id)
				return
			}
			ctr.Db.Model(model).Association("Files").Append(file)

			storagePath, err := ctr.Man.Config.GetStoragePath(pathName)
			if err != nil {
				err = fmt.Errorf("BaseController LookForFileponds: storage path error: %w", err)
				return fParsed, err
			}
			if nestedPath {
				params[0] = storagePath
				err = file.MoveTemp(params...)
			} else {
				err = file.MoveTemp(storagePath)
			}

			ctr.Db.Save(file)

			if err != nil {
				err = fmt.Errorf("BaseController LookForFileponds: moving TempFile error: %w", err)
				return fParsed, err
			}
			fParsed++

		}
	}

	return
}

func (ctr *Controller) AuthGetUser(model interface{}, preload ...string) error {
	if !ctr.Auth.IsIn {
		return fmt.Errorf("Controller AuthGetUser: no user logged in!")
	}

	kind := reflect.ValueOf(model).Type().Kind().String()
	if kind != "ptr" {
		return fmt.Errorf("Controller AuthGetUser: Expected pointer model input! Received non-pointer type.")
	}

	tx := ctr.Db
	for _, pre := range preload {
		tx = tx.Preload(pre)
	}
	preloadErr := tx.Where("id = ?", ctr.Auth.Guid).First(model).Error

	if preloadErr != nil {
		return fmt.Errorf("Controller AuthGetUser getting user (%s) failed: \n%v", ctr.Auth.Username, preloadErr)
	}

	return nil
}

// AppendAuthUser tries to find logged in user and load it to user pointing struct
// if succeed it attaching provided model to user struct
func (ctr *Controller) AppendAuthUser(user interface{}, model interface{}, fieldName ...string) error {
	kind := reflect.ValueOf(model).Type().Kind().String()
	if kind != "ptr" {
		return fmt.Errorf("Controller AppendAuthUser: Expected pointer model input! Received non-pointer type.")
	}

	err := ctr.AuthGetUser(user)
	if err != nil {
		return fmt.Errorf("Controller AppendAuthUser: getting user failed:\n%v", err)
	}

	var field string
	if len(fieldName) == 1 {
		field = fieldName[0]
	} else {
		field = "User"
	}

	return ctr.Db.Model(model).Association(field).Append(user)
}

func (ctr *Controller) GetAltDbConfig() (config *DatabaseConfig) {
	return ctr.Man.Config.DbAlt
}

func (ctr *Controller) CallClient(name string, path string, params url.Values, result interface{}) error {
	client, present := ctr.Man.Clients[name]
	if !present {
		return fmt.Errorf("Controller CallClient: client not found (%s)", name)
	}

	return client.Call(path, params, result)
}

func (ctr *Controller) VerifyApiKey() bool {

	if ctr.Man.Config.ApiKey == nil {
		return false
	}

	return *ctr.Man.Config.ApiKey == ctr.Req.FormSingle("api_key")
}

func (ctr *Controller) QuickSendMessage(text string) error {
	if ctr.Man.Messaging == nil {
		return fmt.Errorf("QuickSendMessage failed: no messenger configured.")
	}

	return ctr.Man.Messaging.QuickSend(text)
}

func (ctr *Controller) SendMessage(msg Message) error {
	if ctr.Man.Messaging == nil {
		return fmt.Errorf("QuickSendMessage failed: no messenger configured.")
	}

	return ctr.Man.Messaging.Send(msg)
}

func (ctr *Controller) SetRequestStartTime(when *time.Time) {
	ctr.Req.startTime = when
}

func (ctr *Controller) FillExecutionTime() {
	ctr.SetCt("_execution_time", ctr.Req.SinceRequestStart().String())
}
