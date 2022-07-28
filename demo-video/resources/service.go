package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const uri = "https://api.chucknorris.io/jokes/random"

func main() {
	http.HandleFunc("/joke", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, getJoke())
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, welcome to testkube demo, use /joke endpoint to get some random joke.")
	})

	log.Fatal(http.ListenAndServe(":8881", nil))
}

func getJoke() string {
	result := struct {
		Value string `json:"value"`
	}{}

	resp, err := http.Get(uri)
	if err != nil {
		return err.Error()
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err.Error()
	}

	return result.Value
}
