package goloabl

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"
)

type Config struct {
	Proxy    Proxy     `json:"proxy"`
	Backends []Backend `json:"backends`
}

// Proxy is a reverse proxy, i.e. a load balancer.
type Proxy struct {
	Port string `json:"port"`
}

// Backend is one of the servers serviced by the Proxy load balancer.
type Backend struct {
	URL    string `json:"url"`
	IsDead bool
	mu     sync.RWMutex
}

var config Config

// Serve serves the load balancer.
func Serve() {
	data, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatal(err.Error())
	}

	json.Unmarshal(data, &config)

	// director must be a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	// Director must not access the provided Request
	// after returning.
	// from: https://pkg.go.dev/net/http/httputil#ReverseProxy
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = ":8081"
	}

	rp := &httputil.ReverseProxy{
		Director: director,
	}

	server := http.Server{
		Addr:    ":8080",
		Handler: rp,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}
