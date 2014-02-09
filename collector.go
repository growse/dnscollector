package main

import (
    "strings"
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
    statsd_address = flag.String("s", "", "statsd_address")
    verbose = flag.Bool("v", false, "verbose")
    snaplen = 65536
)

func flipstringslice(s []string) []string {
    for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
        s[i], s[j] = s[j], s[i]
    }
    if (len(s[0]) == 0) {
        s = s[1:]
    }
    return s
}

func main() {
    expr := "port 53"

    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "usage: %s [ -i interface ] [ -s statsd address ] [ -v ]\n", os.Args[0])
        os.Exit(1)
    }

    flag.Parse()

    if (*statsd_address == "") {
        flag.Usage()
    }

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

    go statsd_pollloop(*statsd_address)

    for pkt, r := h.NextEx(); r >= 0; pkt, r = h.NextEx() {
        if r == 0 {
            // timeout, continue
            continue
        }
        fmt.Println(pkt.Len)
        pkt.Decode()
        if (*verbose) {
            fmt.Println(pkt)
        }
        msg := new(dns.Msg)
        err := msg.Unpack(pkt.Payload)
        if (err == nil) {
            for i := range(msg.Question) {
                domainparts := strings.Split(strings.ToLower(msg.Question[i].Name), ".")
                domainparts = flipstringslice(domainparts)
                domainname := strings.Join(domainparts, ".")
                domainname = strings.Replace(domainname, ".", ",", -1)
                dnstype := strings.ToUpper(dns.TypeToString[msg.Question[i].Qtype])
                if (*verbose) {
                    fmt.Printf("name: %s type: %s\n", domainname, dnstype)
                }
                key := fmt.Sprintf("%s.%s", domainname, dnstype)
                countermap.Lock()
                countermap.m[key]++
                countermap.Unlock()
                bytesmap.Lock()
                bytesmap.m[key]+=pkt.Len
                bytesmap.Unlock()
            }
        }
    }
    fmt.Fprintln(os.Stderr, "tcpdump:", h.Geterror())

}
