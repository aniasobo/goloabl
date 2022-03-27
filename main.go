package goloabl

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
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

// SetIsDead sets the value of IsDead in Backend.
func (backend *Backend) SetIsDead(b bool) {
	backend.mu.Lock()
	backend.IsDead = b
	backend.mu.Unlock()
}

// GetIsDead returns the value of IsDead in Backend.
func (backend *Backend) GetIsDead() bool {
	backend.mu.RLock()
	vitals := backend.IsDead
	backend.mu.RUnlock()
	return vitals
}

// Using sync.Mutex to avoid race conditions caused by
// multiple goroutines accessing variables.
var mu sync.Mutex
var index int = 0

func lbHandler(w http.ResponseWriter, req *http.Request) {
	maxLen := len(config.Backends)

	// implement Round Robin
	mu.Lock()
	backend := config.Backends[index%maxLen]
	if backend.GetIsDead() {
		index++
	}
	targetURL, err := url.Parse(config.Backends[index%maxLen].URL)
	if err != nil {
		log.Fatal(err.Error())
	}
	index++
	mu.Unlock()
	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, e error) {
		log.Printf("%v is dead.", targetURL)
		backend.SetIsDead(true)
		lbHandler(w, req)
	}
	reverseProxy.ServeHTTP(w, req)
}

// IsAlive checks if the backend is alive.
func isAlive(url *url.URL) bool {
	connection, err := net.DialTimeout("tcp", url.Host, time.Minute*1)
	if err != nil {
		log.Printf("Unreachable to %v, error:", url.Host, err.Error())
		return false
	}

	defer connection.Close()
	return true
}

// healthCheck checks the vitals of the backend.
func healthCheck() {
	t := time.NewTicker(time.Minute * 1)
	for {
		select {
		case <-t.C:
			for _, backend := range config.Backends {
				pingURL, err := url.Parse(backend.URL)
				if err != nil {
					log.Fatal(err.Error())
				}
				isAlive := isAlive(pingURL)
				backend.SetIsDead(!isAlive)
				msg := "ok"
				if !isAlive {
					msg = "dead"
				}
				log.Printf("%v checked %v by healthcheck", backend.URL, msg)
			}
		}
	}
}

var config Config

// Serve serves the load balancer.
func Serve() {
	data, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatal(err.Error())
	}

	json.Unmarshal(data, &config)

	go healthCheck()

	// director must be a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	// Director must not access the provided Request
	// after returning.
	// from: https://pkg.go.dev/net/http/httputil#ReverseProxy
	// director := func(req *http.Request) {
	// 	req.URL.Scheme = "http"
	// 	req.URL.Host = ":8081"
	// }

	server := http.Server{
		Addr:    ":" + config.Proxy.Port,
		Handler: http.HandlerFunc(lbHandler),
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}
