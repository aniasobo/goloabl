package goloabl

import (
	"log"
	"net/http"
	"net/http/httputil"
)

func Serve() {
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
