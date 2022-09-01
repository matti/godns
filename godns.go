package godns

import (
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/moby/libnetwork/resolvconf"
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
		// A query for www.microsoft.com returns CNAME and then As
		if dns.TypeToString[answer.Header().Rrtype] != recordType {
			continue
		}
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
	if len(servers) == 0 {
		resolvContents, err := ioutil.ReadFile("/etc/resolv.conf")
		if err != nil {
			panic(err)
		}

		f := resolvconf.File{
			Content: resolvContents,
		}

		for _, nameserver := range resolvconf.GetNameservers(f.Content) {
			servers = append(servers, net.JoinHostPort(nameserver, "53"))
		}
	}
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
