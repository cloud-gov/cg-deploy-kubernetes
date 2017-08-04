package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
var pool *redis.Pool

func handler(w http.ResponseWriter, r *http.Request) {
	client = pool.Get()
	log.Printf("active connections: %d", pool.ActiveCount())
	// Set and check value
	_, err := client.Do("SET", "test", "test")
	if err != nil {
		writeError(w, err)
		return
	}

	value, err := redis.String(client.Do("GET", "test"))
	if err != nil {
		writeError(w, err)
		return
	}
	if value != "test" {
		writeError(w, fmt.Errorf("incorrect value: %s", value))
		return
	}

	_, err = client.Do("DEL", "test")
	if err != nil {
		writeError(w, err)
		return
	}
	client.Close()
}

func newPool(addr string, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   10,
		IdleTimeout: 2 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("AUTH", password); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			threshold := 5 * time.Second
			if time.Since(t) < threshold {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func main() {
	// Get redis32-multinode credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("redis32")
	if len(services) != 1 {
		log.Fatal("redis32 service not found")
	}
	creds := services[0].Credentials

	// Create redis pool
	var err error
	pool = newPool(fmt.Sprintf("%s:%s", creds["hostname"], creds["port"]), creds["password"].(string))
	checkStatus(err)

	// Serve HTTP
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
