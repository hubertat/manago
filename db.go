package manago

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
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
	case "postgres":
		if dbc.config.DisableSsl {
			db, err = gorm.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", dbc.config.User, dbc.config.Pass, dbc.config.Host, dbc.config.Port, dbc.config.Name))
		} else {
			db, err = gorm.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s", dbc.config.Host, dbc.config.Port, dbc.config.User, dbc.config.Name, dbc.config.Pass))
		}

	case "mssql":
		// db, err = gorm.Open("mssql", fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", dbc.config.User, dbc.config.Pass, dbc.config.Host, dbc.config.Port, dbc.config.Name))

		dsn := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s",
			dbc.config.Host, dbc.config.User, dbc.config.Pass, dbc.config.Port, dbc.config.Name)
		if dbc.config.DisableSsl {
			dsn += ";encrypt=disable"
		}
		db, err = gorm.Open("mssql", dsn)

	default:
		db, err = nil, fmt.Errorf("database type not found: %v, cant connect", dbc.config)

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

func (dbc *Db) CopyDb(modelsReflected map[string]reflect.Type, targetDb *Db) error {
	source, err := dbc.Open()
	if err != nil {
		return errors.Join(errors.New("failed to open source db"), err)
	}
	defer source.Close()

	target, err := targetDb.Open()
	if err != nil {
		return errors.Join(errors.New("failed to open target db"), err)
	}
	defer target.Close()

	for _, v := range modelsReflected {
		model := reflect.New(v).Interface()
		rows, err := source.Model(model).Rows()
		if err != nil {
			return errors.Join(errors.New("failed to get rows"), err)
		}

		for rows.Next() {
			err = source.ScanRows(rows, model)
			if err != nil {
				rows.Close()
				return errors.Join(errors.New("failed to scan row"), err)
			}

			result := target.Create(model)
			if result.Error != nil {
				rows.Close()
				return errors.Join(errors.New("failed to create new row in target db"), result.Error)
			}
		}

		rows.Close()
	}

	return nil
}
