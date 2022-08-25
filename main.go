package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type Response struct {
	Error   error
	Server  string
	Status  string
	Records []ResourceRecord
}
type ResourceRecord struct {
	Type   string
	Answer string
	Ttl    uint32
}

func query(recordType string, name string, server string, timeout time.Duration) *Response {
	start := time.Now()
	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}

	response := &Response{
		Server: server,
	}

	c := dns.Client{}

	msg := &dns.Msg{}
	msg.SetQuestion(name, dns.StringToType[recordType])
	msg.RecursionDesired = true

	for _, network := range []string{"udp", "tcp"} {
		c.Net = network
		c.Timeout = timeout - time.Since(start)

		var err error
		msg, _, err = c.Exchange(msg, server)
		if err != nil {
			response.Error = err
			return response
		}

		if msg.Truncated {
			continue
		}

		break
	}

	response.Status = dns.RcodeToString[msg.Rcode]

	for _, answer := range msg.Answer {
		fields := strings.Fields(answer.String())
		response.Records = append(response.Records, ResourceRecord{
			Type:   dns.TypeToString[answer.Header().Rrtype],
			Answer: fields[len(fields)-1],
			Ttl:    answer.Header().Ttl,
		})

	}

	return response
}

func Check(recordType string, name string, timeout time.Duration, servers []string) *Response {
	start := time.Now()

	responses := make(chan *Response)
	for _, server := range servers {
		go func(server string) {
			for i := 0; i < 3; i++ {
				remaining := timeout - time.Since(start)
				response := query(recordType, name, server, remaining)
				if response.Error != nil {
					continue
				}
				responses <- response
				break
			}
		}(server)
	}

	select {
	case <-time.After(timeout):
		return nil
	case r := <-responses:
		return r
	}

}

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

	response := Check(recordType, name, timeout, servers)
	if response == nil {
		log.Println("no response")
		return
	}

	log.Println(response.Server, response.Status, len(response.Records))
	for _, record := range response.Records {
		log.Println(record)
	}
}
