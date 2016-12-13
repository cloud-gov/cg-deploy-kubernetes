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
	// Get elasticsearch24-multinode credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("elasticsearch24")
	if len(services) != 1 {
		log.Fatal("elasticsearch24 service not found")
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

	/*
		Dots in names
		This is a feature that existed pre 2.0 but was turned off
		between 2.0 and 2.3.
		Given that -Dmapper.allow_dots_in_name=true is specified in the
		ES_JAVA_OPTS, it is turned back on.
		More details here: https://www.elastic.co/guide/en/elasticsearch/reference/2.4/dots-in-names.html
	*/
	// Set and check document
	body := `{"server.latency.max": 100}`
	_, err = client.Index().Index("test").Type("test").Id("1").BodyJson(body).Refresh(true).Do()
	checkStatus(err)

	resp, err = client.Get().Index("test").Type("test").Id("1").Do()
	checkStatus(err)

	if !resp.Found {
		log.Fatalf("record not found")
	}

	if string(*resp.Source) != body {
		log.Fatalf("incorrect value: %s", string(*resp.Source))
	}

	// Check the mapping for using dots in names.
	expectedMapping := `{"test":{"mappings":{"test":{"properties":{"server.latency.max":{"type":"long"}}}}}}`
	mapping, err := client.GetMapping().Index("test").Do()
	checkStatus(err)
	jsonMapping, err := json.Marshal(mapping)
	checkStatus(err)

	if string(jsonMapping) != expectedMapping {
		log.Fatalf("incorrect mapping\nvalue: %s\nexpected: %s",
			string(jsonMapping), expectedMapping)
	}

	_, err = client.DeleteIndex("test").Do()
	checkStatus(err)

	// Keep alive
	waitForExit()
}
