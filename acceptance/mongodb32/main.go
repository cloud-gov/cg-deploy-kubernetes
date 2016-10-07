package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/cloudfoundry-community/go-cfenv"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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

type Record struct {
	Key   string
	Value string
}

func main() {
	// Get mongodb32 credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("mongodb32")
	if len(services) != 1 {
		log.Fatal("mongodb32service not found")
	}
	creds := services[0].Credentials

	// Create mongodb client
	session, err := mgo.Dial(fmt.Sprint(creds["uri"]))
	checkStatus(err)
	defer session.Close()

	client := session.DB(fmt.Sprint(creds["dbname"])).C("test")

	// Set and check document
	checkStatus(client.Insert(&Record{Key: "test", Value: "test"}))

	result := Record{}
	checkStatus(client.Find(bson.M{"key": "test"}).One(&result))
	if result.Value != "test" {
		log.Fatalf("incorrect value: %s", result.Value)
	}

	checkStatus(client.Remove(bson.M{"key": "test"}))

	// Keep alive
	waitForExit()
}
