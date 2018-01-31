package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cloudfoundry-community/go-cfenv"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func checkStatus(err error) {
	if err != nil {
		log.Fatalf("error: %s", err.Error())
	}
}

func writeError(w http.ResponseWriter, err error) {
	message, _ := json.Marshal(map[string]string{
		"error": err.Error(),
	})
	w.Write(message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
}

type Record struct {
	Key   string
	Value string
}

var client *mgo.Collection

func handler(w http.ResponseWriter, r *http.Request) {
	// Set and check document
	err := client.Insert(&Record{Key: "test", Value: "test"})
	if err != nil {
		writeError(w, err)
		return
	}

	result := Record{}
	err = client.Find(bson.M{"key": "test"}).One(&result)
	if err != nil {
		writeError(w, err)
		return
	}
	if result.Value != "test" {
		writeError(w, fmt.Errorf("incorrect value: %s", result.Value))
		return
	}

	err = client.Remove(bson.M{"key": "test"})
	if err != nil {
		writeError(w, err)
		return
	}
}

func main() {
	// Get service credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithTag("mongo")
	if len(services) != 1 {
		log.Fatal("mongo service not found")
	}
	creds := services[0].Credentials

	// Create mongodb client
	session, err := mgo.Dial(fmt.Sprint(creds["uri"]))
	checkStatus(err)
	defer session.Close()

	client = session.DB(fmt.Sprint(creds["dbname"])).C("test")

	// Serve HTTP
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
