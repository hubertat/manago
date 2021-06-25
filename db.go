package manago

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
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
		db, err = gorm.Open("sqlite3", "./sqlite/gorm.db")

	case "postgres":
		db, err = gorm.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s", dbc.config.Host, dbc.config.Port, dbc.config.User, dbc.config.Name, dbc.config.Pass))

	case "mssql":
		// db, err = gorm.Open("mssql", fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", dbc.config.User, dbc.config.Pass, dbc.config.Host, dbc.config.Port, dbc.config.Name))
		db, err = gorm.Open("mssql", fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s",
			dbc.config.Host, dbc.config.User, dbc.config.Pass, dbc.config.Port, dbc.config.Name))

	default:
		db, err = nil, fmt.Errorf("Database type not found: %v, cant connect!", dbc.config)

	}

	dbc.DB = db
	return
}

func (dbc *Db) Close() {
	dbc.DB.Close()
}

func (dbc *Db) AutoMigrate(toManage []Manageable) (err error) {

	db, err := dbc.Open()

	if err != nil {
		return fmt.Errorf("models AutoMigrate failed: %w", err)
	}
	defer db.Close()

	for _, typ := range toManage {
		model := typ.MigrateDbModel()
		if model != nil {
			fmt.Printf("Migrating model: %v\n", model)
			db.AutoMigrate(model)
		}
	}

	return
}
