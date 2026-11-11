package traceroute

import (
	"fmt"
	"net"
	"testing"
)

func printHop(hop TracerouteHop) {
	fmt.Printf("%-3d %v (%v)  %v\n", hop.TTL, hop.HostOrAddressString(), hop.AddressString(), hop.ElapsedTime)
}

func TestTraceroute(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Internet and raw-socket access")
	}
	fmt.Printf("Testing synchronous traceroute\n\n")
	out, err := Traceroute("google.com", new(TracerouteOptions))
	if err == nil {
		if len(out.Hops) == 0 {
			t.Errorf("TestTraceroute failed. Expected at least one hop")
		}
	} else {
		t.Errorf("TestTraceroute failed due to an error: %v", err)
	}

	for _, hop := range out.Hops {
		printHop(hop)
	}
	fmt.Println()
}

func TestTraceouteChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Internet and raw-socket access")
	}
	fmt.Printf("Testing asynchronous traceroute\n\n")
	c := make(chan TracerouteHop, 0)
	go func() {
		for {
			hop, ok := <-c
			if !ok {
				fmt.Println()
				return
			}
			printHop(hop)
		}
	}()

	out, err := Traceroute("google.com", new(TracerouteOptions), c)
	if err == nil {
		if len(out.Hops) == 0 {
			t.Errorf("TestTracerouteChannel failed. Expected at least one hop")
		}
	} else {
		t.Errorf("TestTraceroute failed due to an error: %v", err)
	}
}

func TestOptionsDefaults(t *testing.T) {
	options := new(TracerouteOptions)
	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"Port", options.Port(), DEFAULT_PORT},
		{"MaxHops", options.MaxHops(), DEFAULT_MAX_HOPS},
		{"FirstHop", options.FirstHop(), DEFAULT_FIRST_HOP},
		{"TimeoutMs", options.TimeoutMs(), DEFAULT_TIMEOUT_MS},
		{"Retries", options.Retries(), DEFAULT_RETRIES},
		{"PacketSize", options.PacketSize(), DEFAULT_PACKET_SIZE},
	} {
		if tc.got != tc.want {
			t.Errorf("%s() = %d, want default %d", tc.name, tc.got, tc.want)
		}
	}
	if options.IPv6() {
		t.Error("IPv6() default = true, want false")
	}

	options.SetPort(33440)
	options.SetMaxHops(5)
	options.SetFirstHop(2)
	options.SetTimeoutMs(100)
	options.SetRetries(1)
	options.SetPacketSize(64)
	options.SetIPv6(true)
	if options.Port() != 33440 || options.MaxHops() != 5 || options.FirstHop() != 2 ||
		options.TimeoutMs() != 100 || options.Retries() != 1 || options.PacketSize() != 64 || !options.IPv6() {
		t.Errorf("setters not retained: port=%d maxHops=%d firstHop=%d timeoutMs=%d retries=%d packetSize=%d ipv6=%v",
			options.Port(), options.MaxHops(), options.FirstHop(), options.TimeoutMs(), options.Retries(), options.PacketSize(), options.IPv6())
	}

	// Zero means unset: the getter falls back to the default.
	options.SetRetries(0)
	if got := options.Retries(); got != DEFAULT_RETRIES {
		t.Errorf("Retries() after SetRetries(0) = %d, want default %d", got, DEFAULT_RETRIES)
	}
}

func TestHopStrings(t *testing.T) {
	hop := TracerouteHop{Address: net.ParseIP("192.0.2.1")}
	if got := hop.AddressString(); got != "192.0.2.1" {
		t.Errorf("AddressString() = %q, want %q", got, "192.0.2.1")
	}
	if got := hop.HostOrAddressString(); got != "192.0.2.1" {
		t.Errorf("HostOrAddressString() without host = %q, want address", got)
	}
	hop.Host = "router.example.net."
	if got := hop.HostOrAddressString(); got != "router.example.net." {
		t.Errorf("HostOrAddressString() with host = %q, want host", got)
	}
}

func TestDestAddr(t *testing.T) {
	ip, err := destAddr("192.0.2.1", false)
	if err != nil || !ip.Equal(net.ParseIP("192.0.2.1")) {
		t.Errorf("destAddr(ip4, literal) = %v, %v; want 192.0.2.1, nil", ip, err)
	}
	ip, err = destAddr("2001:db8::1", true)
	if err != nil || !ip.Equal(net.ParseIP("2001:db8::1")) {
		t.Errorf("destAddr(ip6, literal) = %v, %v; want 2001:db8::1, nil", ip, err)
	}
	if _, err = destAddr("2001:db8::1", false); err == nil {
		t.Error("destAddr(ip6 literal over ip4) = nil error, want error")
	}
}

func TestSocketAddrSourceSelection(t *testing.T) {
	if src, err := socketAddr(false); err == nil {
		if src.IsLoopback() || src.To4() == nil {
			t.Errorf("socketAddr(false) = %v: loopback or non-IPv4 source", src)
		}
	} else {
		t.Skipf("no usable IPv4 source address: %v", err)
	}
	if src, err := socketAddr(true); err == nil {
		if src.IsLoopback() || src.IsLinkLocalUnicast() || src.To4() != nil {
			t.Errorf("socketAddr(true) = %v: loopback, link-local, or IPv4 source", src)
		}
	} else {
		t.Skipf("no usable global IPv6 source address: %v", err)
	}
}
