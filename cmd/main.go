package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"github.com/lildude/starling-sweep/internal/feeditem"
	"github.com/lildude/starling-sweep/internal/ping"
)

func main() {
	port := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		port = ":" + val
	}

	http.HandleFunc("/_ping", ping.Handler)
	http.HandleFunc("/feed-item", feeditem.Handler)
	fmt.Println("Starting server on port", port)
	log.Fatal(http.ListenAndServe(port, nil)) //#nosec: G114
}
