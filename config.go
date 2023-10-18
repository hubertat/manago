package manago

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hubertat/manago/logging"
)

type FilePath struct {
	Path      string
	ModelName string
	Mime      string
}

type ServerConfig struct {
	Host              string
	Port              uint
	RedirectFromPorts []uint
}

type DatabaseConfig struct {
	Server     string
	SqlitePath string
	Host       string
	Port       uint
	User       string
	Pass       string
	Name       string
	DisableSsl bool
}

type AuthGroup struct {
	UserGroupName string
	Name          string
}

type Config struct {
	Name string

	Server   ServerConfig
	Db       DatabaseConfig
	DbAlt    *DatabaseConfig `json:"db_alt,omitempty"`
	DbTarget *DatabaseConfig `json:"db_target,omitempty"`

	Logging logging.Config

	StoragePaths []FilePath
	MappedPaths  map[string]*FilePath
	DefaultPath  string
	TmpPath      string

	TemplatesPath   string
	StaticPath      string
	WebStaticPath   string
	ForceLiveStatic bool

	AuthGroups []AuthGroup
	MappedAuth map[string]*AuthGroup

	DevSkipMiddleware bool

	ApiKey  *string
	Clients map[string]Client

	SlackHook *string `json:",omitempty"`

	AppVariables map[string]string

	SessionLifetimeHours uint `json:"session_lifetime_hours,omitempty"`
}

func (c *Config) Load(input string) (err error) {

	c.FillDefaults()

	err = c.ReadFile(input)
	if err != nil {
		err = fmt.Errorf("ERROR Config Load: %w", err)
	}
	return
}

func (c *Config) FillDefaults() {
	c.Server.Port = 8080

	c.Db.Server = "sqlite"
	c.Db.SqlitePath = "./sqlite/gorm.db"

	c.StaticPath = "./static/"
	c.TemplatesPath = "templates"
	c.WebStaticPath = "web"

	c.DefaultPath = "./files/"

	c.Name = "Default APP name"
}

func (c *Config) ReadFile(fPath string) error {
	cFile, err := os.ReadFile(fPath)
	if err != nil {
		return fmt.Errorf("config ReadFile [%v]: %w", fPath, err)
	}

	err = json.Unmarshal([]byte(cFile), c)
	if err != nil {
		return fmt.Errorf("config parse json: %w", err)
	}

	var notEmpty bool

	c.MappedPaths = make(map[string]*FilePath)
	for _, sPath := range c.StoragePaths {
		if len(sPath.ModelName) > 0 {
			_, notEmpty = c.MappedPaths[sPath.ModelName]
			if !notEmpty {
				c.MappedPaths[sPath.ModelName] = &sPath
			}
		}
	}

	c.MappedAuth = make(map[string]*AuthGroup)
	for _, aG := range c.AuthGroups {
		if len(aG.Name) > 0 {
			_, notEmpty = c.MappedAuth[aG.Name]
			if !notEmpty {
				c.MappedAuth[aG.Name] = &aG
			}
		}
	}
	return nil
}

func (c *Config) GetStoragePath(ins ...string) (string, error) {
	var model, mime string
	switch len(ins) {
	case 0:
		return c.DefaultPath, nil
	case 1:
		model = ins[0]
		mime = ""
	default:
		model = ins[0]
		mime = ins[1]
	}

	paths := make(map[string]string)
	for _, everyP := range c.StoragePaths {
		if everyP.ModelName == model {
			paths[everyP.Mime] = everyP.Path
		}
	}

	if len(paths) == 0 {
		// return c.DefaultPath, fmt.Errorf("config GetStoragePath: no paths found for %s", model)
		return c.DefaultPath, nil
	}

	path, found := paths[mime]
	if found {
		return path, nil
	}

	// return c.DefaultPath, fmt.Errorf("config GetStoragePath: no path found for %s/%s", model, mime)
	return c.DefaultPath, nil
}
