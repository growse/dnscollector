dnscollector
============

A small daemon that counts DNS queries and publishes them to statsd. It uses libpcap to listen for traffic on the specified interface on port 53, and pushes a counter out to the speicified statsd address every 10s.

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

`dnscollector` takes two arguments:

    dnscollector -i <interface> -s <statsd_address>:<statsd_port>

It will then listen for TCP and UDP packets on port 53 on `interface` and publish periodic statistics to `statsd_address` on `statsd_port`.
