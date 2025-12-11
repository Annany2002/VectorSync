package main

import (
	"net/http"

	"github.com/Annany2002/vector-sync/internal/db"
	"github.com/Annany2002/vector-sync/internal/logger"
)

var (
	log = logger.NewLogger()
)

func main(){
	server := &http.Server{
		Addr: ":6309",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Infof("%s %s", r.Method, r.URL.Path)
			w.Write([]byte("Hello from VectorSync!"))
		}),
	}

	// Connect to database
	db, err := db.Connect()
	if err != nil {
		log.Errorf("Error connecting to database: %v", err)
	}
	defer db.Close()

	log.Infof("Server started on :6309")
	log.Errorf("Server failed to start: %v", server.ListenAndServe())
}