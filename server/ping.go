package server

import (
	"runtime"
	"strings"
	"time"

	"github.com/go-ping/ping"

	"fpdns/lib"
)

type pingStatus struct {
	PingAt     time.Time
	Statistics *ping.Statistics
}

var (
	nameserverPingStatus = map[string][]*pingStatus{}
)

func startMonitorNameservers() {
	nameservers := resolver.Nameservers()
	for i := 0; i < len(nameservers); i++ {
		nameserver := strings.Split(nameservers[i], ":")[0]
		nameserverPingStatus[nameserver] = []*pingStatus{nil, nil, nil}
		go pingLoop(nameserver)
	}
}

func pingLoop(server string) {
	for {
		doping(server)
		time.Sleep(time.Second * 10)
	}
}

func doping(server string) {
	// fmt.Println("ping", server)
	pinger, err := ping.NewPinger(server)
	if err != nil {
		lib.AppLog().Errorf("ping server [%s] faild: %s\n", server, err)
		return
	}
	pinger.Count = 10
	pinger.Timeout = time.Second * 30
	if runtime.GOOS == "linux" {
		pinger.SetPrivileged(true)
	}
	pinger.Run()                 // blocks until finished
	stats := pinger.Statistics() // get send/receive/rtt stats
	// fmt.Printf("%+#v\n", stats)

	nameserverPingStatus[server][0] = &pingStatus{
		time.Now(),
		stats,
	}
}
