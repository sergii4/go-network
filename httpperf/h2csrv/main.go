package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	h2s := &http2.Server{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_health" {
			w.Write([]byte("OK"))
			return
		}
		_, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, "can't read body", http.StatusBadRequest)
			return
		}
		w.Write([]byte(payload))
		fmt.Println("http vesrion", r.Proto)
		//		fmt.Fprintf(w, "Hello, %v, http: %v", r.URL.Path, r.TLS == nil)
	})

	server := &http.Server{
		Addr:         "0.0.0.0:8080",
		Handler:      h2c.NewHandler(handler, h2s),
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 10 * time.Minute,
	}

	fmt.Printf("Listening [0.0.0.0:8080]...\n")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}

const (
	payload = `
	{"foo": "bar"}

`
)


