package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
)

var port = flag.Int("port", 8080, "http port")

func init() {
	flag.Parse()
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%s\n", dump)
	})

	fmt.Printf("Listening on :%d port", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
