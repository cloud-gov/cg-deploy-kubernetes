package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"

	"github.com/cloudfoundry-community/go-cfenv"
	"gopkg.in/olivere/elastic.v3"
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
	// Get elasticsearch23-multinode credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("elasticsearch23")
	if len(services) != 1 {
		log.Fatal("elasticsearch23 service not found")
	}
	creds := services[0].Credentials

	// Create elasticsearch client
	client, err := elastic.NewClient(
		elastic.SetURL("http://"+creds["hostname"].(string)+":"+creds["port"].(string)),
		elastic.SetBasicAuth(creds["username"].(string), creds["password"].(string)),
		elastic.SetSniff(false),
	)
	checkStatus(err)

	// Set and check document
	record := Record{Key: "key", Value: "value"}
	_, err = client.Index().Index("test").Type("test").Id("1").BodyJson(record).Refresh(true).Do()
	checkStatus(err)

	resp, err := client.Get().Index("test").Type("test").Id("1").Do()
	checkStatus(err)

	if !resp.Found {
		log.Fatalf("record not found")
	}

	result := Record{}
	err = json.Unmarshal(*resp.Source, &result)
	checkStatus(err)
	if result.Value != "value" {
		log.Fatalf("incorrect value: %s", result.Value)
	}

	_, err = client.DeleteIndex("test").Do()
	checkStatus(err)

	// Keep alive
	waitForExit()
}
