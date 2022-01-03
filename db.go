package manago

import (
	"fmt"
	"reflect"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

type Db struct {
	DB *gorm.DB

	config DatabaseConfig
}

func (dbc *Db) Check(config DatabaseConfig) (err error) {

	dbc.config = config

	_, err = dbc.Open()
	return
}

func (dbc *Db) Open() (db *gorm.DB, err error) {
	switch dbc.config.Server {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open("./sqlite/gorm.db"))

	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s", dbc.config.Host, dbc.config.Port, dbc.config.User, dbc.config.Name, dbc.config.Pass)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})

	case "mssql":
		dsn := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s",
			dbc.config.Host, dbc.config.User, dbc.config.Pass, dbc.config.Port, dbc.config.Name)
		db, err = gorm.Open(sqlserver.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	default:
		db, err = nil, fmt.Errorf("database type not found: %v, cant connect!", dbc.config)

	}

	dbc.DB = db
	return
}

func (dbc *Db) Close() {
}

func (dbc *Db) AutoMigrate(modelsReflected map[string]reflect.Type) (err error) {

	db, err := dbc.Open()

	if err != nil {
		return fmt.Errorf("models AutoMigrate failed: %w", err)
	}

	for _, v := range modelsReflected {
		model := reflect.New(v).Interface()
		fmt.Printf("Migrating model: %v\n", v)
		db.AutoMigrate(model)
	}

	return
}
