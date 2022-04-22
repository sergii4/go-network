package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {

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
		Handler:      handler,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 10 * time.Minute,
	}

	fmt.Printf("Listening [0.0.0.0:8080]...\n")

	log.Fatal(server.ListenAndServeTLS("../server.crt", "../server.key"))

}

const (
	payload = `
	{"foo": "bar"}

`
)
