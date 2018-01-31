package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/cloudfoundry-community/go-cfenv"
	"gopkg.in/olivere/elastic.v5"
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
	resp, err := client.ClusterState().Do(context.Background())
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
	resp, err := client.NodesInfo().Do(context.Background())
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
	resp, err := client.ClusterHealth().Do(context.Background())
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
	_, err := client.Index().Index("test").Type("test").Id("1").BodyJson(record).Refresh("true").Do(context.Background())
	if err != nil {
		writeError(w, err)
		return
	}

	resp, err := client.Get().Index("test").Type("test").Id("1").Do(context.Background())
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
		return
	}

	_, err = client.DeleteIndex("test").Do(context.Background())
	if err != nil {
		writeError(w, err)
		return
	}
}

func main() {
	// Get elasticsearch56-multinode credentials
	env, _ := cfenv.Current()
	services, _ := env.Services.WithLabel("elasticsearch56")
	if len(services) != 1 {
		log.Fatal("elasticsearch56 service not found")
	}
	creds := services[0].Credentials

	// Create elasticsearch client
	var err error

	client, err = elastic.NewClient(
		elastic.SetURL("http://"+creds["hostname"].(string)+":"+creds["port"].(string)),
		elastic.SetSniff(false),
	)
	if err == nil {
		log.Fatalf("error: must not be able to connect without credentials")
	}

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
