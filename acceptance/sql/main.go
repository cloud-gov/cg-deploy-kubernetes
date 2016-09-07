package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	// "strings"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/cloudfoundry-community/go-cfenv"
)

func checkStatus(err error) {
	if err != nil {
		log.Fatalf("error: %s", err.Error())
	}
}

func waitForExit() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
	os.Exit(0)
}

func main() {
	driver := os.Getenv("SQL_DRIVER")
	if driver == "" {
		log.Fatal("environment variable SQL_DRIVER not found")
	}

	service := os.Getenv("SQL_SERVICE")
	if service == "" {
		log.Fatal("environment variable SQL_SERVICE not found")
	}

	// Get credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel(service)
	if len(services) != 1 {
		log.Fatalf("%s service not found", service)
	}
	creds := services[0].Credentials

	// Create client
	// uri := strings.Replace(fmt.Sprint(creds["uri"]), "mysql://", "", -1)
	uri := fmt.Sprint(creds["uri"])
	db, err := sql.Open(driver, uri)
	checkStatus(err)

	_, err = db.Exec("CREATE TABLE acceptance (id INTEGER, value TEXT)")
	checkStatus(err)

	_, err = db.Exec("INSERT INTO acceptance VALUES (1, 'acceptance')")
	checkStatus(err)

	var value int64
	row := db.QueryRow("SELECT value FROM acceptance WHERE id = $1", 1)
	err = row.Scan(&value)
	checkStatus(err)
	if value != 1 {
		log.Fatalf("incorrect value: %d", value)
	}

	_, err = db.Exec("DROP TABLE acceptance")
	checkStatus(err)

	// Keep alive
	waitForExit()
}
