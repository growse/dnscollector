dnscollector
============

A small daemon that counts DNS queries and publishes them to statsd. It uses libpcap to listen for traffic on the specified interface on port 53, and pushes a counter out to the speicified statsd address every 10s.

Build
-----

To build, you'll need golang installed. Then simply:

    GOPATH=`pwd` go get main && go build -o dnscollector main

in the root directory, and the executable should be output in the root directory called `dnscollector`.
