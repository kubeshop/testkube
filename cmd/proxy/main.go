package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

var port = flag.Int("port", 8080, "HTTP Proxy port")
var apiPort = flag.Int("apiPort", 8088, "HTTP API port")

func init() {
	flag.Parse()
}

func main() {
	fmt.Printf("Serve on :%d\n\n", *port)
	http.HandleFunc("/", proxyPass)
	panic(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

func proxyPass(res http.ResponseWriter, req *http.Request) {
	body, _ := ioutil.ReadAll(req.Body)
	prefix := fmt.Sprintf("/api/v1/namespaces/testkube/services/testkube-api-server:%d/proxy", *apiPort)
	req.URL.Path = strings.Replace(req.URL.Path, prefix, "", -1)
	req, _ = http.NewRequest(req.Method, req.URL.String(), bytes.NewReader(body))

	url, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", *apiPort))
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ServeHTTP(res, req)
}
