package main

import (
	"log"
	"net/http"

	"github.com/Annany2002/vector-sync/internal/db"
)

func main(){
	server := &http.Server{
		Addr: ":6309",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.Method, r.URL.Path)
			w.Write([]byte("Hello from VectorSync!"))
		}),
	}

	// Connect to database
	db, err := db.Connect()
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer db.Close()

	log.Fatal(server.ListenAndServe())
}