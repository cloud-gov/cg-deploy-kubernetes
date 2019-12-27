package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/garyburd/redigo/redis"
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

var client redis.Conn

func handler(w http.ResponseWriter, r *http.Request) {
	// Set and check value
	_, err := client.Do("SET", "test", "test")
	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}

	value, err := redis.String(client.Do("GET", "test"))
	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}
	if value != "test" {
		err := fmt.Errorf("incorrect value: %s", value)
		writeError(w, err)
		checkStatus(err)
		return
	}

	_, err = client.Do("DEL", "test")
	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}
}

func main() {
	// Get redis28-multinode credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("redis28")
	if len(services) != 1 {
		log.Fatal("redis28 service not found")
	}
	creds := services[0].Credentials

	// Create redis client
	var err error
	client, err = redis.Dial("tcp", fmt.Sprintf("%s:%s", creds["hostname"], creds["port"]))
	checkStatus(err)
	defer client.Close()

	_, err = client.Do("AUTH", creds["password"].(string))
	checkStatus(err)

	// Serve HTTP
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
