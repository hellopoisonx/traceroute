# Traceroute in Go

A traceroute library written in Go.

Must be run as sudo on OS X (and others)?

## Library

```go
options := traceroute.TracerouteOptions{}
options.SetIPv6(true)
result, err := traceroute.Traceroute("example.com", &options)
```

The traceroute.Traceroute() function accepts a domain name and an options struct and returns a TracerouteResult struct that holds an array of TracerouteHop structs.

IPv4 is used unless the options ask for IPv6 (`options.SetIPv6(true)`), in which case the
destination is resolved and probed over IPv6 with ICMPv6 replies.

## Resources

Useful resources:

* http://en.wikipedia.org/wiki/Traceroute
* http://tools.ietf.org/html/rfc792
* http://en.wikipedia.org/wiki/Internet_Control_Message_Protocol

## Notes

* https://code.google.com/p/go/source/browse/src/pkg/net/ipraw_test.go
* http://godoc.org/code.google.com/p/go.net/ipv4
* https://pkg.go.dev/golang.org/x/net/ipv6
