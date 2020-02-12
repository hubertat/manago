package main

import (
	"log"
	"time"
	"flag"

	"manago"
	"manago/cmd/sample/controllers"	
)


func main() {
	flagMigrate := flag.Bool("migrate", false, "Perform AutoMigrate (gorm)")
	flagPort := flag.Uint("port", 0, "Port to serve http")
	flagHost := flag.String("host", "", "Hostname/ip to serve http from")
	flagConfigFile := flag.String("config", "config.json", "Path to the configuration file (json)")

	flag.Parse()

	conf := app.Config{}
    err := conf.Load(*flagConfigFile)
    if err != nil {
    	log.Panicf("Creating configuration error: %v", err)
    }

	
    if *flagPort != 0 {
    	conf.Server.Port = *flagPort
    }

    if *flagHost != "" {
    	conf.Server.Host = *flagHost
    }

    log.Printf("Config loaded:\n%+v\n", conf)

	var allModels = []interface{}{
		// models.User{},
	}

	var allCtrs = []interface{}{
		// controllers.UserController{},
		controllers.Sample{},
	}

    gotech, err := app.New(conf, allCtrs, allModels)
    if err != nil {
    	log.Fatalf("Application New failed:\n%v", err)
    }

    if *flagMigrate {
    	log.Print("Starting AutoMigrate for every model")
	    err = gotech.Migrate()
	    if err != nil {
	    	log.Print(err)
	    }
    }
    
    log.Print(gotech.Start())

    for {
    	time.Sleep(100 * time.Second)
    }
}

