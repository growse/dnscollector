package main

import (
    "os"
    "time"
    "sync"
    "net"
    "bytes"
    "strconv"
    "fmt"
)

var statsd_namespace_prefix string
var con *net.UDPConn

func statsd_dispatcher(data map[string]int) (error) {
    var record string
    var count int
    for record, count = range data {
        var buffer bytes.Buffer
        buffer.WriteString(statsd_namespace_prefix)
        buffer.WriteString(record)
        buffer.WriteString(":")
        buffer.WriteString(strconv.Itoa(count))
        buffer.WriteString("|c\n")
        con.Write(buffer.Bytes())
    }
    return nil
}

var countermap = struct{
    sync.RWMutex
    m map[string]int
}{m: make(map[string]int)}

var snapshot map[string]int

func statsd_pollloop(address string) {
    serverAddr, err := net.ResolveUDPAddr("udp", address)
    con, err = net.DialUDP("udp", nil, serverAddr)
    countermap.m = make(map[string]int)
    if err != nil {
        fmt.Printf("Error connecting to %s: %s", address, err)
        os.Exit(1)
    } else {
        for {
            time.Sleep(10 * time.Second)
            snapshot := make(map[string]int)
            countermap.RLock()

            for k,v := range(countermap.m) {
                snapshot[k] = v
            }

            countermap.RUnlock()

            go statsd_dispatcher(snapshot)
            countermap.m = make(map[string]int)
        }
    }
}
