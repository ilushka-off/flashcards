package main

import (
	"io"
	"log"
	"net/http"
)

func main() {

	handler := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "I'm a live")
	}

	http.HandleFunc(`/healtz`, handler)
	err := http.ListenAndServe(":8082", nil)
	if err != nil {
		log.Fatal(err)
	}
}
