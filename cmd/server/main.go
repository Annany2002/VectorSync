package main

import (
	"log"
	"net/http"
)

func main(){
	server := &http.Server{
		Addr: ":6309",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.Method, r.URL.Path)
			w.Write([]byte("Hello from VectorSync!"))
		}),
	}
	log.Fatal(server.ListenAndServe())
}