package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cloudfoundry-community/go-cfenv"
	"gopkg.in/olivere/elastic.v3"
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

var client *elastic.Client

func state(w http.ResponseWriter, r *http.Request) {
	resp, err := client.ClusterState().Do()
	if err != nil {
		writeError(w, err)
		return
	}

	jresp, err := json.Marshal(resp)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jresp)
}

func nodes(w http.ResponseWriter, r *http.Request) {
	resp, err := client.NodesInfo().Do()
	if err != nil {
		writeError(w, err)
		return
	}

	jresp, err := json.Marshal(resp)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jresp)
}

func health(w http.ResponseWriter, r *http.Request) {
	resp, err := client.ClusterHealth().Do()
	if err != nil {
		writeError(w, err)
		return
	}

	jresp, err := json.Marshal(resp)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jresp)
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Set and check document
	record := Record{Key: "key", Value: "value"}
	_, err := client.Index().Index("test").Type("test").Id("1").BodyJson(record).Refresh(true).Do()
	if err != nil {
		writeError(w, err)
		return
	}

	resp, err := client.Get().Index("test").Type("test").Id("1").Do()
	if err != nil {
		writeError(w, err)
		return
	}

	if !resp.Found {
		writeError(w, err)
		return
	}

	result := Record{}
	err = json.Unmarshal(*resp.Source, &result)
	if err != nil {
		writeError(w, err)
		return
	}

	if result.Value != "value" {
		writeError(w, fmt.Errorf("incorrect value: %s", result.Value))
	}

	_, err = client.DeleteIndex("test").Do()
	if err != nil {
		writeError(w, err)
		return
	}

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
	if err != nil {
		writeError(w, err)
		return
	}

	resp, err = client.Get().Index("test").Type("test").Id("1").Do()
	if err != nil {
		writeError(w, err)
		return
	}

	if !resp.Found {
		writeError(w, errors.New("record not found"))
		log.Fatalf("record not found")
	}

	if string(*resp.Source) != body {
		writeError(w, fmt.Errorf("incorrect value: %s", string(*resp.Source)))
	}

	// Check the mapping for using dots in names.
	expectedMapping := `{"test":{"mappings":{"test":{"properties":{"server.latency.max":{"type":"long"}}}}}}`
	mapping, err := client.GetMapping().Index("test").Do()
	if err != nil {
		writeError(w, err)
		return
	}

	jsonMapping, err := json.Marshal(mapping)
	if err != nil {
		writeError(w, err)
		return
	}

	if string(jsonMapping) != expectedMapping {
		writeError(w, fmt.Errorf("incorrect mapping\nvalue: %s\nexpected: %s", string(jsonMapping), expectedMapping))
	}

	_, err = client.DeleteIndex("test").Do()
	if err != nil {
		writeError(w, err)
		return
	}
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
	var err error
	client, err = elastic.NewClient(
		elastic.SetURL("http://"+creds["hostname"].(string)+":"+creds["port"].(string)),
		elastic.SetBasicAuth(creds["username"].(string), creds["password"].(string)),
		elastic.SetSniff(false),
	)
	checkStatus(err)

	// Serve HTTP
	http.HandleFunc("/", handler)
	http.HandleFunc("/cluster-health", health)
	http.HandleFunc("/cluster-nodes", nodes)
	http.HandleFunc("/cluster-state", state)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
