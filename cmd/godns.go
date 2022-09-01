package main

import (
	"flag"
	"log"
	"time"

	"github.com/matti/godns"
)

func main() {
	var recordType string
	var timeout time.Duration
	var name string
	var servers []string
	flag.StringVar(&recordType, "recordType", "A", "recordType")
	flag.DurationVar(&timeout, "timeout", time.Second, "timeout")
	flag.Parse()

	name = flag.Args()[0]
	servers = flag.Args()[1:]

	response := godns.Check(recordType, name, timeout, servers)
	if response == nil {
		log.Println("no response from", servers)
		return
	}

	log.Println(response.Server, response.Status, len(response.Records))
	for _, record := range response.Records {
		log.Println(record)
	}
}
