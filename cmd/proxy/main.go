package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var port = flag.Int("port", 8080, "HTTP Proxy port")
var apiPort = flag.Int("apiPort", 8088, "HTTP API port")
var namespace = flag.String("namespace", "testkube", "Testkube installation namespace")

func init() {
	flag.Parse()
}

func main() {
	fmt.Printf("Serve on :%d ==> ", *port)
	fmt.Printf("Proxy to :%d\n", *apiPort)
	fmt.Printf("Testkube installed in Kubernetes namespace: %s\n\n", *namespace)

	http.HandleFunc("/", proxyPass)
	panic(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

type DebugTransport struct{}

func (DebugTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	b, err := httputil.DumpRequestOut(r, false)
	if err != nil {
		fmt.Printf("error: %+v\n", err)
		return nil, err
	}
	fmt.Println(string(b))

	return http.DefaultTransport.RoundTrip(r)
}

func proxyPass(res http.ResponseWriter, req *http.Request) {
	fmt.Printf("\n-------------\n")
	body, _ := io.ReadAll(req.Body)
	fmt.Printf("%s\n", body)

	prefix := fmt.Sprintf("/api/v1/namespaces/%s/services/testkube-api-server:%d/proxy", *namespace, *apiPort)
	req.URL.Path = strings.Replace(req.URL.Path, prefix, "", -1)

	newReq := req.Clone(req.Context())
	newReq.Body = io.NopCloser(bytes.NewReader(body))

	url, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", *apiPort))
	proxy := httputil.NewSingleHostReverseProxy(url)
	if _, ok := os.LookupEnv("DEBUG"); ok {
		proxy.Transport = DebugTransport{}
	}

	proxy.ServeHTTP(res, newReq)
}
