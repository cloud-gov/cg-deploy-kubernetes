package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"

	"github.com/cloudfoundry-community/go-cfenv"
	"gopkg.in/olivere/elastic.v2"
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
	// Get elasticsearch17-multinode credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("elasticsearch17")
	if len(services) != 1 {
		log.Fatal("elasticsearch17 service not found")
	}
	creds := services[0].Credentials

	// Create elasticsearch client
	client, err := elastic.NewClient(elastic.SetURL(creds["uri"]))

	// Set and check document
	record := Record{Key: "key", Value: "value"}
	_, err = client.Index().Index("test").Type("test").Id("1").BodyJson(record).Refresh(true).Do()
	checkStatus(err)

	query := elastic.NewTermQuery("key", "value")
	results, err := client.Search().Index("test").Query(query).Do()
	checkStatus(err)

	if results.Hits.TotalHits != 1 {
		log.Fatalf("should find exactly one record; found %d", results.Hits.TotalHits)
	}

	result := Record{}
	err = json.Unmarshal(*results.Hits.Hits[0].Source, &result)
	checkStatus(err)
	if record.Value != "value" {
		log.Fatalf("incorrect value: %s", result.Value)
	}

	_, err = client.DeleteIndex("test").Do()
	checkStatus(err)

	// Keep alive
	waitForExit()
}
