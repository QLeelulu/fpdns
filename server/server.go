package server

import (
	"errors"
	"math/rand"
	_ "net/http/pprof"
	"strings"
	"sync/atomic"
	"time"

	"fpdns/lib"

	"github.com/miekg/dns"
)

type ServerConfig struct {
	ConfDir  string // dns配置所在的目录
	Addr     string // 监听的ip和端口， 例如 :53 或者 127.0.0.1:53
	HttpAddr string // http服务监听的ip和端口， 例如 :8666 或者 127.0.0.1:8666

	CacheTTL int // 缓存DNS解析结果的过期时间，单位秒。

	LogFile  string // 日志文件路径，为空则输出到标准输出
	LogLevel int    // 日志打印级别。ERROR:1, WARN:2, NOTICE:3, LOG:4, DEBUG:5, NO:0 。
}

var (
	sc ServerConfig

	resolvConfFile string
	resolver       *lib.Resolver

	// 自定义配置的域名列表
	rrCache map[string]map[[2]uint16][]dns.RR
	// 远程解析的域名列表
	resolvCache *lib.MemoryCache

	logInstance lib.Logger

	monitorCount  int64 //统计计算
	currentQPS    float64
	perTotalCount int64

	ErrCNAMELoop = errors.New("CNAME loop")
)

const (
	qpsInterval = 10
)

func init() {
	rrCache = map[string]map[[2]uint16][]dns.RR{}

	rand.Seed(time.Now().UTC().UnixNano())

}

func StartServer(c ServerConfig) {

	sc = c

	if len(sc.LogFile) > 0 {
		lib.SetLogFile(sc.LogFile)
		lib.UseStdout(false)
	}
	logInstance = lib.AppLog()
	logInstance.SetLogLevel(sc.LogLevel)

	// resolvCache = &MemoryCache{
	// 	Backend:  make(map[[2]uint16]map[string]Mesg, 0),
	// 	Expire:   time.Duration(cacheTTL) * time.Second,
	// 	Maxcount: 0,
	// }
	var err error
	resolvCache, err = lib.NewMemoryCache(sc.CacheTTL)
	if err != nil {
		logInstance.Fatalf("init cache error: %s", err)
	}

	loadConf(sc.ConfDir)
	initResolver()
	listenAndServe()
	go InitHTTP(sc.HttpAddr)
	monitorQPS()
}

func monitorQPS() {
	//定期刷新监控的最新信息
	go func() {
		for {
			time.Sleep(qpsInterval * time.Second)
			currentQPS = float64((monitorCount - perTotalCount)) / float64(qpsInterval)
			perTotalCount = monitorCount
		}
	}()
}

func listenAndServe() {
	tcpHandler := dns.NewServeMux()
	tcpHandler.HandleFunc(".", handleTCPRequest)

	udpHandler := dns.NewServeMux()
	udpHandler.HandleFunc(".", handleUDPRequest)

	server := &dns.Server{Addr: sc.Addr, Net: "udp"}
	server.TsigSecret = map[string]string{"axfr.": "so6ZGir4GPAqINNh9U5c3A=="}
	server.Handler = udpHandler
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			logInstance.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()

	tcpserver := &dns.Server{Addr: sc.Addr, Net: "tcp"}
	tcpserver.TsigSecret = map[string]string{"axfr.": "so6ZGir4GPAqINNh9U5c3A=="}
	tcpserver.Handler = tcpHandler
	go func() {
		err := tcpserver.ListenAndServe()
		if err != nil {
			logInstance.Fatalf("Failed to set tcp listener %s\n", err.Error())
		}
	}()
}

func handleRequest(netType string, w dns.ResponseWriter, r *dns.Msg) {
	atomic.AddInt64(&monitorCount, 1)

	if len(r.Question) < 1 {
		logInstance.Errorf("question len is 0: %+v", r)
		dns.HandleFailed(w, r)
		return
	}

	q := r.Question[0]

	if logInstance.LogLevel() <= lib.LOG_LEVEL_DEBUG {
		logInstance.Debugf("%s query [type:%s, class:%s, name:%s] from %s.", netType,
			dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass],
			q.Name, w.RemoteAddr())
	}

	m, err := queryDnsResult(netType, r, 0)

	if err != nil {
		logInstance.Errorf("resolve [type:%s, class:%s, name:%s] query from [%s] error: %s",
			dns.TypeToString[q.Qtype],
			dns.ClassToString[q.Qclass],
			q.Name, w.RemoteAddr(), err)
		dns.HandleFailed(w, r)
		return
	}

	if r.IsTsig() != nil {
		if w.TsigStatus() == nil {
			// *Msg r has an TSIG record and it was validated
			m.SetTsig("axfr.", dns.HmacMD5, 300, time.Now().Unix())
		} else {
			// *Msg r has an TSIG records and it was not valided
			logInstance.Errorf("ERROR TSIG Status: %s", w.TsigStatus().Error())
		}
	}
	m.Compress = true
	w.WriteMsg(m)
}

func handleTCPRequest(w dns.ResponseWriter, r *dns.Msg) {
	handleRequest("tcp", w, r)
}

func handleUDPRequest(w dns.ResponseWriter, r *dns.Msg) {
	handleRequest("udp", w, r)
}

func initResolver() {
	if resolvConfFile == "" {
		resolvConfFile = "/etc/resolv.conf"
	}
	clientConfig, err := dns.ClientConfigFromFile(resolvConfFile)
	if err != nil {
		logInstance.Errorf("%s is not a valid resolv.conf file\n", resolvConfFile)
		panic(err)
	}
	resolver = &lib.Resolver{
		Config: clientConfig,
	}
	startMonitorNameservers()
}

func getFromResolver(netType string, r *dns.Msg) (message *dns.Msg, err error) {
	q := r.Question[0]
	q.Name = strings.ToLower(q.Name)

	cacheMessage, cacheErr := resolvCache.Get(q)
	if cacheErr == nil && cacheMessage != nil {
		message = cacheMessage
		return
	}
	message, err = resolver.Lookup(netType, r)
	if err != nil {
		// 如果之前有缓存结果，则返回之前的缓存结果
		if cacheErr == lib.KeyExpiredError && cacheMessage != nil {
			message = cacheMessage
			err = nil
		}
		return
	} else if message != nil {
		resolvCache.Set(q, message)
	}
	return
}

func isIPQuery(r *dns.Msg) bool {
	if len(r.Question) < 1 {
		return false
	}
	q := r.Question[0]
	// 只处理IPV4的查询
	if q.Qclass == dns.ClassINET && q.Qtype == dns.TypeA {
		return true
	}
	return false
}

// @deep: 预防无限递归
func queryDnsResult(netType string, r *dns.Msg, deep int) (*dns.Msg, error) {
	if deep > 5 {
		return nil, ErrCNAMELoop
	}
	m := new(dns.Msg)
	q := r.Question[0]

	_isIPQuery := isIPQuery(r)
	getOk := false
	name := strings.ToLower(q.Name)
	rrsAll, ok := rrCache[name]

	// wildcard records
	if !ok && _isIPQuery { // ip查询才处理泛解析
		// 泛解析查找
		nameArr := strings.Split(name, ".")
		for i := 0; i < len(nameArr)-3; i++ {
			wildDomain := "*." + strings.Join(nameArr[i+1:], ".")
			rrsAll, ok = rrCache[wildDomain]
			if ok {
				// rrs[0].Header().Name = name
				break
			}
		}
	}

	if ok {
		rrs, ok := rrsAll[[2]uint16{q.Qclass, q.Qtype}]
		if ok && len(rrs) > 0 {
			loadBalancing(rrs)
		}
		// CNAME
		// 没找到记录的情况下，非CNAME查询则查一下是否有CNAME记录
		if !ok && q.Qtype != dns.TypeCNAME { // q.Qtype 为CNAME的时候，直接返回
			rrs, ok = rrsAll[[2]uint16{q.Qclass, dns.TypeCNAME}]

			if ok && len(rrs) > 0 {
				r2 := new(dns.Msg)
				rrCNAMME := rrs[0].(*dns.CNAME)
				if strings.ToUpper(rrCNAMME.Target) == "DIRECT." {
					logInstance.Debugf("will DIRECT resole [type:%s, class:%s, name:%s] from unstream resolver",
						dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass], q.Name)
					goto DirectGetFromResolver
				}
				// 查询CNAME目标的DNS解析记录
				r2.Question = []dns.Question{
					dns.Question{
						Name:   rrCNAMME.Target,
						Qtype:  dns.TypeA,
						Qclass: dns.ClassINET,
					},
				}
				deep++
				mCNAME, err := queryDnsResult(netType, r2, deep)
				if err != nil {
					return nil, err
				}
				if mCNAME != nil && len(mCNAME.Answer) > 0 {
					rrs = append(rrs, mCNAME.Answer...)
				}
			}
		}

		if ok && len(rrs) > 0 {
			rrs[0].Header().Name = name // 泛解析记录的时候需要这样特殊处理
			m.Answer = rrs
			m.SetReply(r)
			getOk = true
			logInstance.Debugf("resole [type:%s, class:%s, name:%s] from local config",
				dns.TypeToString[q.Qtype], dns.ClassToString[q.Qclass], q.Name)
		}
	}

DirectGetFromResolver:
	if !getOk {
		var err error
		m, err = getFromResolver(netType, r)
		if err != nil {
			return nil, err
		} else if m != nil {
			m.Id = r.Id
		}
	}
	return m, nil
}

func loadBalancing(rrs []dns.RR) {
	rand.Shuffle(len(rrs), func(i, j int) {
		rrs[i], rrs[j] = rrs[j], rrs[i]
	})
}
