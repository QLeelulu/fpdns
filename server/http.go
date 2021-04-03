package server

import (
	"fmt"
	"net/http"
	"time"

	"fpdns/lib"
)

// InitHTTP 初始化HTTP服务
func InitHTTP(addr string) {
	http.HandleFunc("/debug", debugHandler)

	http.HandleFunc("/reload_conf", reloadConfHandler)

	lib.AppLog().Debugln("start http server at ", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		lib.AppLog().Errorln("start http failed: ", err)
	}
}

func debugHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Local config cache len:%d\n", len(rrCache))
	// for k, v := range rrCache {
	// 	fmt.Fprintf(w, "\t[class:%s, type:%s]: %d\n",
	// 		dns.ClassToString[k[0]], dns.TypeToString[k[1]], len(v))
	// }

	fmt.Fprintf(w, "\nResolved cache len: %d\n", resolvCache.Length())
	fmt.Fprintf(w, "\n\nDNS Query QPS: %f\n", currentQPS)

	fmt.Fprintf(w, "\n\nDNS Nameservers Ping: \n")
	for k, v := range nameserverPingStatus {
		fmt.Fprintf(w, "\t%s: \n", k)
		if v[0] != nil {
			fmt.Fprintf(w, "\t\t [%s]: send:%d, recv:%d, loss:%.1f, avgRtt:%dms \n",
				v[0].PingAt.Format("2006-01-02 15:04:05"),
				v[0].Statistics.PacketsSent,
				v[0].Statistics.PacketsRecv,
				v[0].Statistics.PacketLoss,
				v[0].Statistics.AvgRtt/time.Millisecond)
		}
	}
}

func reloadConfHandler(w http.ResponseWriter, r *http.Request) {
	add, del, change := reloadDNSConf()
	fmt.Fprintf(w, "reload dns conf done.\n\n")

	printDNSConfChangeInfo(w, "add", add)
	printDNSConfChangeInfo(w, "delete", del)
	printDNSConfChangeInfo(w, "change", change)
}

func printDNSConfChangeInfo(w http.ResponseWriter, t string, list []string) {
	fmt.Fprintf(w, "dns conf %s: %d\n", t, len(list))
	for _, val := range list {
		fmt.Fprintf(w, "\t %s %v\n", t, val)
	}
	fmt.Fprintf(w, "\n")
}
