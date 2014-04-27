package main

import (
    "log"
    "net"
    "strings"
    "flag"
    "fmt"
    "os"
    "github.com/growse/pcap"
    "github.com/miekg/dns"
    "github.com/abh/geoip"
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
    expr = ""
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
        log.Fatal(fmt.Sprintf("tcpdump fatal error: %s", err))
    }

    derr := h.SetDirection("out")
    if derr != nil {
        fmt.Println("tcpdump:", derr)
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
    geoasn, err := geoip.OpenType(geoip.GEOIP_ASNUM_EDITION)
    if err != nil {
        log.Fatal("Could not open geoip database")
    }
    geocountry, err := geoip.OpenType(geoip.GEOIP_COUNTRY_EDITION)
    if err != nil {
        log.Fatal("Could not open geoip database")
    }

    for pkt, r := h.NextEx(); r >= 0; pkt, r = h.NextEx() {
        if r == 0 {
            // timeout, continue
            continue
        }
        pkt.Decode()
        if len(pkt.Headers) > 0 {
            ipHdr, ipOk := pkt.Headers[0].(*pcap.Iphdr)
            if ipOk {
                //ip := fmt.Sprintf("%d.%d.%d.%d", ipHdr.DestIp[0],ipHdr.DestIp[1],ipHdr.DestIp[2],ipHdr.DestIp[3])
                ip := net.IPv4(
                    ipHdr.DestIp[0],
                    ipHdr.DestIp[1],
                    ipHdr.DestIp[2],
                    ipHdr.DestIp[3],
                ).String()
                name, _ := geoasn.GetName(ip)
                fmt.Println(name)
                country, _ := geocountry.GetCountry(ip)
                fmt.Println(country)
            }
        }
        msg := new(dns.Msg)
        err := msg.Unpack(pkt.Payload)
        // We only want packets which have been successfully unpacked 
        // and have at least one answer
        if (err == nil && len(msg.Answer) > 0) {
            if (*verbose) {
                fmt.Println(pkt)
            }
            for i := range(msg.Question) {
                domainparts := strings.Split(strings.ToLower(msg.Question[i].Name), ".")
                domainparts = flipstringslice(domainparts)
                domainname := strings.Join(domainparts, ".")
                domainname = strings.Replace(domainname, ".", ">", -1)
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
