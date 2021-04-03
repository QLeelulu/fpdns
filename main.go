package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"fpdns/server"
)

var (
	confDir  string
	addr     string
	httpAddr string

	cacheTTL int

	logFile  string
	logLevel int
)

func parseFlag() {
	flag.StringVar(&confDir, "conf_dir", "", "directory included config files. 包含DNS配置的目录")
	flag.StringVar(&addr, "addr", ":53", "ip addresses to listen on. 监听的ip和端口， 例如 :53 或者 127.0.0.1:53")
	flag.StringVar(&httpAddr, "http_addr", ":8666", "http services ip addresses to listen on. http服务监听的ip和端口， 例如 :8666 或者 127.0.0.1:8666")

	flag.IntVar(&cacheTTL, "cache_ttl", 30, "seconds cache TTL. 缓存DNS解析结果的过期时间，单位秒。默认30秒。")
	flag.IntVar(&logLevel, "log_level", 5, "log level. 日志打印级别。 NO:0, ERROR:1, WARN:2, NOTICE:3, LOG:4, DEBUG:5 。默认5.")
	flag.StringVar(&logFile, "log_file", "", "log file to send write to instead of stdout - has to be a file, not directory. 日志文件路径，默认输出到标准输出")

	flag.Parse()

	if confDir == "" {
		fmt.Println("配置目录 conf_dir 参数必须指定")
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	parseFlag()

	sc := server.ServerConfig{}
	sc.Addr = addr
	sc.CacheTTL = cacheTTL
	sc.ConfDir = confDir
	sc.HttpAddr = httpAddr
	sc.LogFile = logFile
	sc.LogLevel = logLevel

	server.StartServer(sc)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGHUP)
	func() {
		<-c
		os.Exit(0)
	}()
}
