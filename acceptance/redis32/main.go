package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/go-redis/redis/v7"
)

func checkStatus(err error) {
	if err != nil {
		log.Fatalf("error: %s", err.Error())
	}
}

func writeError(w http.ResponseWriter, err error) {
	log.Printf("There was an error, %s\n", err.Error())
	message, _ := json.Marshal(map[string]string{
		"error": err.Error(),
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(message)
}

func newConnection() *redis.Client {
	// Get redis32-multinode credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("redis32")
	if len(services) != 1 {
		log.Fatal("redis32 service not found")
	}
	creds := services[0].Credentials

	// set the timeouts so we can have some more definitive answers.
	newClient := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", creds["hostname"], creds["port"]),
		Password:     creds["password"].(string),
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	_, err := newClient.Ping().Result()
	checkStatus(err)

	return newClient
}

func testSetGetDelete(w http.ResponseWriter, r *http.Request) {
	client := newConnection()
	defer client.Close()

	// Set and check value
	err := client.Set("test", "test", 0).Err()
	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}

	value, err := client.Get("test").Result()
	if err != nil {
		checkStatus(err)
		writeError(w, err)
		return
	}
	if value != "test" {
		err := fmt.Errorf("incorrect value: %s", value)
		writeError(w, err)
		checkStatus(err)
		return
	}

	err = client.Del("test").Err()
	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func info(w http.ResponseWriter, r *http.Request) {
	client := newConnection()
	defer client.Close()

	parameter := r.URL.Query().Get("s")

	if parameter == "" {
		parameter = "all"
	}

	infoString, err := client.Info().Result()
	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}

	infoMap := make(map[string]string)

	for _, line := range strings.Split(infoString, "\r\n") {
		part := strings.Split(line, ":")
		if len(part) == 2 {
			infoMap[part[0]] = part[1]
		}
	}

	jresp, err := json.Marshal(infoMap)

	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jresp)

}

func configGet(w http.ResponseWriter, r *http.Request) {
	client := newConnection()
	defer client.Close()

	parameter := r.URL.Query().Get("p")

	if parameter == "" {
		parameter = "*"
	}

	primaryConfig, err := client.ConfigGet(parameter).Result()
	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}

	jresp, err := json.Marshal(primaryConfig)

	if err != nil {
		writeError(w, err)
		checkStatus(err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jresp)

}

func main() {
	// Serve HTTP
	http.HandleFunc("/", testSetGetDelete)
	http.HandleFunc("/config-get", configGet)
	http.HandleFunc("/info", info)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
