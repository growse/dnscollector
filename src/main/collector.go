package main

import (
    "flag"
    "fmt"
    "os"
    "github.com/miekg/pcap"
    "github.com/miekg/dns"
)

const (
    TYPE_IP  = 0x0800
    TYPE_ARP = 0x0806
    TYPE_IP6 = 0x86DD

    IP_ICMP = 1
    IP_INIP = 4
    IP_TCP  = 6
    IP_UDP  = 17
)

var (
    device  = flag.String("i", "", "interface")
    snaplen = 65536
)

func main() {
    expr := "port 53"

    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "usage: %s [ -i interface ]\n", os.Args[0])
        os.Exit(1)
    }

    flag.Parse()

    if *device == "" {
        devs, err := pcap.FindAllDevs()
        if err != "" {
            fmt.Fprintln(os.Stderr, "tcpdump: couldn't find any devices:", err)
        }
        if 0 == len(devs) {
            flag.Usage()
        }
        *device = devs[0].Name
    }

    h, err := pcap.OpenLive(*device, int32(snaplen), true, 500)
    if h == nil {
        fmt.Fprintf(os.Stderr, "tcpdump:", err)
        return
    }

    ferr := h.SetFilter(expr)
    if ferr != nil {
        fmt.Println("tcpdump:", ferr)
    }

    hostname, err := os.Hostname()

    if err == nil {
        statsd_namespace_prefix = fmt.Sprintf("%s.dnscollector.",hostname)
    }

    go statsd_pollloop("localhost:5000")

    for pkt, r := h.NextEx(); r >= 0; pkt, r = h.NextEx() {
        if r == 0 {
            // timeout, continue
            continue
        }
        pkt.Decode()
        fmt.Println(pkt)
        msg := new(dns.Msg)
        err := msg.Unpack(pkt.Payload)
        if (err == nil) {
            for i := range(msg.Question) {
                fmt.Printf("name: %s type: %s\n", msg.Question[i].Name,dns.TypeToString[msg.Question[i].Qtype])
                key := fmt.Sprintf("%s%s", msg.Question[i].Name, dns.TypeToString[msg.Question[i].Qtype])
                countermap.Lock()
                countermap.m[key]++
                countermap.Unlock()
            }
        }
    }
    fmt.Fprintln(os.Stderr, "tcpdump:", h.Geterror())

}
