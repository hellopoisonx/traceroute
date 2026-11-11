# Traceroute in Go

A traceroute library written in Go.

## Library

```go
options := traceroute.TracerouteOptions{}
options.SetIPv6(true)
result, err := traceroute.Traceroute("example.com", &options)
```

The traceroute.Traceroute() function accepts a domain name and an options struct and returns a TracerouteResult struct that holds an array of TracerouteHop structs.

IPv4 is used unless the options ask for IPv6 (`options.SetIPv6(true)`), in which case the
destination is resolved and probed over IPv6 with ICMPv6 replies.

Two probe modes are supported. By default the probes are UDP datagrams sent to
`options.Port()` (default 33434): intermediate hops answer with ICMP Time
Exceeded and the destination with ICMP Port Unreachable. `options.SetICMP(true)`
switches to ICMP Echo Request probes, answered by an Echo Reply from the
destination (`Port()` does not apply in that mode).

## Testing

`go test ./...` runs only offline tests. The end-to-end traces against
`google.com` (Internet and raw-socket access required) are opt-in and take
minutes on a filtered path:

```bash
TRACEROUTE_E2E=1 go test -run 'TestTraceroute|TestTraceouteChannel' -v .
```

## Resources

Useful resources:

* http://en.wikipedia.org/wiki/Traceroute
* http://tools.ietf.org/html/rfc792
* http://en.wikipedia.org/wiki/Internet_Control_Message_Protocol

## Notes

* https://code.google.com/p/go/source/browse/src/pkg/net/ipraw_test.go
* http://godoc.org/code.google.com/p/go.net/ipv4
* https://pkg.go.dev/golang.org/x/net/ipv6
