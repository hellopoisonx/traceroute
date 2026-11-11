// Package traceroute provides functions for executing a traceroute to a remote
// host.
package traceroute

import (
	"context"
	"errors"
	"net"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	DEFAULT_PORT        = 33434
	DEFAULT_MAX_HOPS    = 64
	DEFAULT_FIRST_HOP   = 1
	DEFAULT_TIMEOUT_MS  = 500
	DEFAULT_RETRIES     = 3
	DEFAULT_PACKET_SIZE = 52
)

func socketAddr(useIPv6 bool) (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if useIPv6 {
			// A link-local source cannot reach off-link destinations.
			if ipnet.IP.To4() == nil && !ipnet.IP.IsLinkLocalUnicast() {
				return ipnet.IP, nil
			}
		} else if ip := ipnet.IP.To4(); ip != nil {
			return ip, nil
		}
	}
	return nil, errors.New("You do not appear to be connected to the Internet")
}

func destAddr(dest string, useIPv6 bool) (net.IP, error) {
	network := "ip4"
	if useIPv6 {
		network = "ip6"
	}
	ipAddr, err := net.ResolveIPAddr(network, dest)
	if err != nil {
		return nil, err
	}
	return ipAddr.IP, nil
}

type TracerouteOptions struct {
	port       int
	maxHops    int
	firstHop   int
	timeoutMs  int
	retries    int
	packetSize int
	ipv6       bool
}

func (options *TracerouteOptions) Port() int {
	if options.port == 0 {
		options.port = DEFAULT_PORT
	}
	return options.port
}

func (options *TracerouteOptions) SetPort(port int) {
	options.port = port
}

func (options *TracerouteOptions) MaxHops() int {
	if options.maxHops == 0 {
		options.maxHops = DEFAULT_MAX_HOPS
	}
	return options.maxHops
}

func (options *TracerouteOptions) SetMaxHops(maxHops int) {
	options.maxHops = maxHops
}

func (options *TracerouteOptions) FirstHop() int {
	if options.firstHop == 0 {
		options.firstHop = DEFAULT_FIRST_HOP
	}
	return options.firstHop
}

func (options *TracerouteOptions) SetFirstHop(firstHop int) {
	options.firstHop = firstHop
}

func (options *TracerouteOptions) TimeoutMs() int {
	if options.timeoutMs == 0 {
		options.timeoutMs = DEFAULT_TIMEOUT_MS
	}
	return options.timeoutMs
}

func (options *TracerouteOptions) SetTimeoutMs(timeoutMs int) {
	options.timeoutMs = timeoutMs
}

func (options *TracerouteOptions) Retries() int {
	if options.retries == 0 {
		options.retries = DEFAULT_RETRIES
	}
	return options.retries
}

func (options *TracerouteOptions) SetRetries(retries int) {
	options.retries = retries
}

func (options *TracerouteOptions) PacketSize() int {
	if options.packetSize == 0 {
		options.packetSize = DEFAULT_PACKET_SIZE
	}
	return options.packetSize
}

func (options *TracerouteOptions) SetPacketSize(packetSize int) {
	options.packetSize = packetSize
}

func (options *TracerouteOptions) IPv6() bool {
	return options.ipv6
}

func (options *TracerouteOptions) SetIPv6(ipv6 bool) {
	options.ipv6 = ipv6
}

type TracerouteHop struct {
	Success     bool
	Address     net.IP
	Host        string
	N           int
	ElapsedTime time.Duration
	TTL         int
}

func (hop *TracerouteHop) AddressString() string {
	return hop.Address.String()
}

func (hop *TracerouteHop) HostOrAddressString() string {
	hostOrAddr := hop.AddressString()
	if hop.Host != "" {
		hostOrAddr = hop.Host
	}
	return hostOrAddr
}

type TracerouteResult struct {
	DestinationAddress net.IP
	Hops               []TracerouteHop
}

func notify(hop TracerouteHop, channels []chan TracerouteHop) {
	for _, c := range channels {
		c <- hop
	}
}

func closeNotify(channels []chan TracerouteHop) {
	for _, c := range channels {
		close(c)
	}
}

// Traceroute uses the given dest (hostname) and options to execute a traceroute
// from your machine to the remote host.
//
// Outbound packets are UDP packets and inbound packets are ICMP (ICMPv6 when
// options.IPv6() is set).
//
// Returns a TracerouteResult which contains an array of hops. Each hop includes
// the elapsed time and its IP address.
func Traceroute(dest string, options *TracerouteOptions, c ...chan TracerouteHop) (result TracerouteResult, err error) {
	result.Hops = []TracerouteHop{}
	destAddr, err := destAddr(dest, options.IPv6())
	if err != nil {
		return
	}
	result.DestinationAddress = destAddr
	socketAddr, err := socketAddr(options.IPv6())
	if err != nil {
		return
	}

	timeout := time.Duration(options.TimeoutMs()) * time.Millisecond

	// IPv6 answers with ICMPv6 and uses the hop limit where IPv4 uses the TTL.
	recvNetwork, sendNetwork := "ip4:icmp", "udp4"
	if options.IPv6() {
		recvNetwork, sendNetwork = "ip6:ipv6-icmp", "udp6"
	}

	ttl := options.FirstHop()
	retry := 0
	for {
		start := time.Now()

		recvSocket, err := net.ListenIP(recvNetwork, &net.IPAddr{IP: socketAddr})
		if err != nil {
			return result, err
		}
		defer recvSocket.Close()

		if err := recvSocket.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return result, err
		}

		sendSocket, err := net.ListenUDP(sendNetwork, nil)
		if err != nil {
			return result, err
		}
		defer sendSocket.Close()

		if options.IPv6() {
			err = ipv6.NewPacketConn(sendSocket).SetHopLimit(ttl)
		} else {
			err = ipv4.NewPacketConn(sendSocket).SetTTL(ttl)
		}
		if err != nil {
			return result, err
		}

		if _, err := sendSocket.WriteToUDP([]byte{0x0}, &net.UDPAddr{IP: destAddr, Port: options.Port()}); err != nil {
			return result, err
		}

		p := make([]byte, options.PacketSize())
		n, from, err := recvSocket.ReadFromIP(p)
		elapsed := time.Since(start)
		if err == nil {
			currAddr := from.IP

			hop := TracerouteHop{Success: true, Address: currAddr, N: n, ElapsedTime: elapsed, TTL: ttl}

			resolveCtx, cancel := context.WithTimeout(context.Background(), timeout)
			currHost, err := net.DefaultResolver.LookupAddr(resolveCtx, hop.AddressString())
			cancel()
			if err == nil {
				hop.Host = currHost[0]
			}

			notify(hop, c)

			result.Hops = append(result.Hops, hop)

			ttl += 1
			retry = 0

			if ttl > options.MaxHops() || currAddr.Equal(destAddr) {
				closeNotify(c)
				return result, nil
			}
		} else {
			retry += 1
			if retry > options.Retries() {
				notify(TracerouteHop{Success: false, TTL: ttl}, c)
				ttl += 1
				retry = 0
			}

			if ttl > options.MaxHops() {
				closeNotify(c)
				return result, nil
			}
		}

	}
}
