package main

import (
    "bytes"
    "fmt"
    "net"
    "os"
    "strconv"
    "sync"
    "time"
)

var statsd_namespace_prefix string
var con *net.UDPConn

func statsd_dispatcher(counterdata map[string]uint32, bytesdata map[string]uint32) error {
    var record string
    var count uint32
    for record, count = range counterdata {
        var buffer bytes.Buffer
        buffer.WriteString(statsd_namespace_prefix)
        buffer.WriteString(record)
        buffer.WriteString("_count")
        buffer.WriteString(":")
        buffer.WriteString(strconv.FormatUint(uint64(count), 10))
        buffer.WriteString("|c\n")
        con.Write(buffer.Bytes())
    }
    for record, count = range bytesdata {
        var buffer bytes.Buffer
        buffer.WriteString(statsd_namespace_prefix)
        buffer.WriteString(record)
        buffer.WriteString("_bytes")
        buffer.WriteString(":")
        buffer.WriteString(strconv.FormatUint(uint64(count), 10))
        buffer.WriteString("|c\n")
        con.Write(buffer.Bytes())
    }
    return nil
}

var countermap = struct {
    sync.RWMutex
    m   map[string]uint32
}{m: make(map[string]uint32)}

var bytesmap = struct {
    sync.RWMutex
    m   map[string]uint32
}{m: make(map[string]uint32)}

var countersnapshot map[string]uint32
var bytessnapshot map[string]uint32

func statsd_pollloop(address string) {
    serverAddr, err := net.ResolveUDPAddr("udp", address)
    con, err = net.DialUDP("udp", nil, serverAddr)
    countermap.m = make(map[string]uint32)
    bytesmap.m = make(map[string]uint32)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error connecting to %s: %s\n", address, err)
        os.Exit(1)
    } else {
        for {
            time.Sleep(10 * time.Second)
            countersnapshot := make(map[string]uint32)
            bytessnapshot := make(map[string]uint32)
            countermap.RLock()
            bytesmap.RLock()

            for k, v := range countermap.m {
                countersnapshot[k] = v
            }
            for k, v := range bytesmap.m {
                bytessnapshot[k] = v
            }

            countermap.RUnlock()
            bytesmap.RUnlock()

            go statsd_dispatcher(countersnapshot, bytessnapshot)
            countermap.m = make(map[string]uint32)
            bytesmap.m = make(map[string]uint32)
        }
    }
}
