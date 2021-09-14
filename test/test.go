package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", HelloServer)
	http.HandleFunc("/hello", HelloServer1)
	http.ListenAndServe(":3030", nil)
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "1234" {
		w.WriteHeader(401)
		return
	}
	fmt.Fprintf(w, "Hello, %s!", "3030")
}
func HelloServer1(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "1234" {
		w.WriteHeader(401)
		return
	}
	fmt.Fprintf(w, "Hello 123, %s!", "3030")
}
