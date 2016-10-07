package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/garyburd/redigo/redis"
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
	// Get redis28-multinode credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("redis28")
	if len(services) != 1 {
		log.Fatal("redis28service not found")
	}
	creds := services[0].Credentials

	// Create redis client
	client, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", creds["hostname"], creds["port"]))
	checkStatus(err)
	defer client.Close()

	// Authenticate redis
	_, err = client.Do("AUTH", creds["password"])
	checkStatus(err)

	// Set and check value
	_, err = client.Do("SET", "test", "test")
	checkStatus(err)

	value, err := redis.String(client.Do("GET", "test"))
	checkStatus(err)
	if value != "test" {
		log.Fatalf("incorrect value: %s", value)
	}

	_, err = client.Do("DEL", "test")
	checkStatus(err)

	// Keep alive
	waitForExit()
}
