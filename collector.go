package main

import (
    "flag"
    "net/url"
    "fmt"
    "github.com/abh/geoip"
    "github.com/growse/pcap"
    "github.com/miekg/dns"
    "log"
    "net"
    "os"
    "strings"
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
    device         = flag.String("i", "", "interface")
    statsd_address = flag.String("s", "", "statsd_address")
    verbose        = flag.Bool("v", false, "verbose")
    dnsport        = flag.Int("p", 53, "dnsport")
    snaplen        = 65536
)

func flipstringslice(s []string) []string {
    for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
        s[i], s[j] = s[j], s[i]
    }
    if len(s[0]) == 0 {
        s = s[1:]
    }
    return s
}

func main() {
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "usage: %s [ -p dnsport ] [ -i interface ] [ -s statsd address ] [ -v ]\n", os.Args[0])
        os.Exit(1)
    }

    flag.Parse()
    expr := fmt.Sprintf("port %d", *dnsport)

    if *statsd_address == "" {
        flag.Usage()
    }

    if *device == "" {
        devs, err := pcap.FindAllDevs()
        fmt.Println(devs)
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
        statsd_namespace_prefix = fmt.Sprintf("%s.dnscollector.", hostname)
    }

    go statsd_pollloop(*statsd_address)
    geoasn, err := geoip.OpenType(geoip.GEOIP_ASNUM_EDITION)
    if err != nil {
        log.Fatal("Could not open geoip database: GEOIP_ASNUM_EDITION")
    }
    geocountry, err := geoip.OpenType(geoip.GEOIP_COUNTRY_EDITION)
    if err != nil {
        log.Fatal("Could not open geoip database: GEOIP_COUNTRY_EDITION")
    }
    geoasn6, err := geoip.OpenType(geoip.GEOIP_ASNUM_EDITION_V6)
    if err != nil {
        log.Fatal("Could not open geoip database: GEOIP_ASNUM_EDITION_V6")
    }
    geocountry6, err := geoip.OpenType(geoip.GEOIP_COUNTRY_EDITION_V6)
    if err != nil {
        log.Fatal("Could not open geoip database: GEOIP_COUNTRY_EDITION_V6")
    }
    var name, country string
    var ipOk bool
    var ipHdr *pcap.Iphdr
    var ip6Hdr *pcap.Ip6hdr
    for pkt, r := h.NextEx(); r >= 0; pkt, r = h.NextEx() {
        if r == 0 {
            // timeout, continue
            continue
        }
        pkt.Decode()
        msg := new(dns.Msg)
        err := msg.Unpack(pkt.Payload)
        // We only want packets which have been successfully unpacked
        // and have at least one answer
        if err == nil && len(msg.Answer) > 0 {
            if *verbose {
                fmt.Println(pkt)
            }
            if len(pkt.Headers) > 0 {
                //IPv4
                if ipHdr, ipOk = pkt.Headers[0].(*pcap.Iphdr); ipOk {
                    ip := net.IPv4(
                        ipHdr.DestIp[0],
                        ipHdr.DestIp[1],
                        ipHdr.DestIp[2],
                        ipHdr.DestIp[3],
                    ).String()
                    name, _ = geoasn.GetName(ip)
                    country, _ = geocountry.GetCountry(ip)
                } else if ip6Hdr, ipOk = pkt.Headers[0].(*pcap.Ip6hdr); ipOk {
                    //IPv6
                    ip := net.IP(ip6Hdr.DestIp).String()
                    name, _ = geoasn6.GetNameV6(ip)
                    country, _ = geocountry6.GetCountry_v6(ip)
                }
            }
            for i := range msg.Question {
                domainparts := strings.Split(strings.ToLower(msg.Question[i].Name), ".")
                domainparts = flipstringslice(domainparts)
                domainname := strings.Join(domainparts, ".")
                domainname = strings.Replace(domainname, ".", ">", -1)
                dnstype := strings.ToUpper(dns.TypeToString[msg.Question[i].Qtype])
                if len(name) == 0 {
                    name = "none"
                }
                if len(country) == 0 {
                    country = "none"
                }
                name = url.QueryEscape(name)

                name = strings.Replace(name, ".", "%2E", -1)
                country = strings.Replace(country, ".", "%2E", -1)
                key := fmt.Sprintf("%s.%s.%s.%s", domainname, name, country, dnstype)
                if *verbose {
                    fmt.Println(key)
                }
                countermap.Lock()
                countermap.m[key]++
                countermap.Unlock()
                bytesmap.Lock()
                bytesmap.m[key] += pkt.Len
                bytesmap.Unlock()
            }
        }
    }
    fmt.Fprintln(os.Stderr, "tcpdump:", h.Geterror())

}
