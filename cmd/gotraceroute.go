package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/aeden/traceroute"
)

func printHop(hop traceroute.TracerouteHop) {
	if hop.Success {
		fmt.Printf("%-3d %v (%v)  %v\n", hop.TTL, hop.HostOrAddressString(), hop.AddressString(), hop.ElapsedTime)
	} else {
		fmt.Printf("%-3d *\n", hop.TTL)
	}
}

func main() {
	m := flag.Int("m", traceroute.DEFAULT_MAX_HOPS, `Set the max time-to-live (max number of hops) used in outgoing probe packets (default is 64)`)
	f := flag.Int("f", traceroute.DEFAULT_FIRST_HOP, `Set the first used time-to-live, e.g. the first hop (default is 1)`)
	q := flag.Int("q", 1, `Set the number of probes per "ttl" to nqueries (default is one probe).`)
	six := flag.Bool("6", false, `Use IPv6`)

	flag.Parse()
	host := flag.Arg(0)
	if host == "" {
		flag.Usage()
		os.Exit(2)
	}
	options := traceroute.TracerouteOptions{}
	options.SetRetries(*q - 1)
	options.SetMaxHops(*m + 1)
	options.SetFirstHop(*f)
	options.SetIPv6(*six)

	network := "ip4"
	if options.IPv6() {
		network = "ip6"
	}
	ipAddr, err := net.ResolveIPAddr(network, host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot resolve %v over %v: %v\n", host, network, err)
		os.Exit(1)
	}

	fmt.Printf("traceroute to %v (%v), %v hops max, %v byte packets\n", host, ipAddr, options.MaxHops(), options.PacketSize())

	c := make(chan traceroute.TracerouteHop, 0)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for hop := range c {
			printHop(hop)
		}
		fmt.Println()
	}()

	_, err = traceroute.Traceroute(host, &options, c)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		// Traceroute only closes its notify channels on non-error returns.
		close(c)
	}
	<-done
}
