package main

import (
	"flag"
	"fmt"
	"http"
	"log"
	"pusher"
	"time"
)

func main() {
	mode := flag.Int("mode", 0, "0: Broadcast, 1: LIFO, 2: FILO")
	flag.Parse()

	config := pusher.DefaultConfiguration
	config.GCInterval = 0 // disable garbage collecting
	config.ConcurrencyMode = *mode

	// Create a new pusher context, where all publisher and subscriber
	// locations are statically mapped to "static"-channel.
	push := pusher.New(pusher.StaticAcceptor("static"), config)
	http.Handle("/pub", push.PublisherHandler)
	http.Handle("/sub", push.SubscriberHandler)
	http.Handle("/", http.FileServer("www/", "/"))

	// Create the "static"-channel explictly, since AllowChannelCreation is false
	// in the DefaultConfiguration.
	channel, _ := push.Channel("static")

	go func() {
		i := 0
		for {
			channel.PublishString(fmt.Sprintf("--- Greetings from the server #%d ---", i), false)
			time.Sleep(10e9)
			i++
		}
	}()

	log.Print("Tune your browser tab(s) to http://localhost:8080/")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
