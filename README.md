dnscollector
============

A small daemon that counts DNS queries and publishes them, along with the total byte size of the responses to statsd. It uses libpcap to listen for traffic on the specified interface on port 53, and pushes a counter out to the speicified statsd address every 10s.

For each query name / type combination that is seen, `dnscollector` will publish two counter values every 10 seconds:

    hostname.dnscollector.com,example,www.A_count:10|c
    hostname.dnscollector.com,example,www.A_bytes:493|c

It will also replace the '.' character in the query name with a comma - this is to help with keeping the name as a single graphite key.

Dependencies
------------

`dnscollector` uses two other go projects, `github.com/miekg/pcap` and `github.com/miekg/dns`. `github.com/miegk/pcap` has a system dependency on `libpcap0.8` so you'll need that installed. On Debian/Ubuntu this should be as simple as:

    apt-get install libpcap0.8

Build
-----

To build, you'll need golang installed. Then simply create a go project directory structure (with pkg, bin and src) folders, do a 

    GOPATH=`pwd` go get github.com/growse/dnscollector

Followed by

    GOPATH=`pwd` go install github.com/growse/dnscollector

in the go project directory, and the executable should be output in the bin directory.

Usage
-----

`dnscollector` takes three arguments:

    dnscollector -i <interface> -s <statsd_address>:<statsd_port> -v

It will then listen for TCP and UDP packets on port 53 on `interface` and publish periodic statistics to `statsd_address` on `statsd_port`. If `-v` is specified, it will print to `stdout` details of packets that it sees and the parsed DNS information from them.
