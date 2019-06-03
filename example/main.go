package main

import (
	"log"
	"net/http"
	"elfinder"
	"fmt"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./elf/"))))
	mux.Handle("/connector",http.HandlerFunc(elfHandler))
	fmt.Println("Listen on :8000")
	log.Fatal(http.ListenAndServe(":8000", mux))
}

func elfHandler(w http.ResponseWriter,r *http.Request){

	con := elfinder.NewElFinderConnector([]elfinder.Volume{&elfinder.DefaultVolume})
	con.ServeHTTP(w,r)
}