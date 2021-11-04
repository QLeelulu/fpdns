# fpdns

Fast Private DNS，提供自定义的DNS记录配置和DNS解析缓存。

特性：

- A记录
- CNAME
- 泛解析
- DNS负载均衡
- 缓存DNS解析结果
- 上游同时多DNS Server查询

## 各系统测试情况

- Linux: 已稳定运行3年多
- Darwin: 已开发测试
- Windows: 未测试

## 安装

直接下载对应系统的执行文件 [latest release](https://github.com/QLeelulu/fpdns/releases/latest)。

如果你熟悉go语言，希望自行编译，可以执行如下命令：

```
./build.sh
```

会生成可执行文件 `fpdns` 在`bin`目录.

## 运行

```
./fpdns -conf_dir ./conf
```

命令行参数：

```
Usage of ./fpdns:
  -addr string
    	监听的ip和端口， 例如 :53 或者 127.0.0.1:53 (default ":53")
  -cache_ttl int
    	缓存DNS解析结果的过期时间，单位秒。默认30秒。 (default 30)
  -conf_dir string
    	读取配置的目录
  -http_addr string
    	http服务监听的ip和端口， 例如 :8666 或者 127.0.0.1:8666 (default ":8666")
  -log_file string
    	日志文件路径，默认输出到标准输出
  -log_level int
    	日志打印级别。ERROR:1, WARN:2, NOTICE:3, LOG:4, DEBUG:5, NO:0 。默认5. (default 5)
```

## 配置文件

### resolv.conf

会先从命令行参数`-conf_dir`指定的配置目录中读取`resolv.conf`文件，如果文件不存在，则从`/etc/resolv.conf`读取。

`resolv.conf`文件里面配置多个DNS Server的时候，fpdns会从上往下每隔1秒逐个请求，并返回最早响应的解析结果。

- 第0秒请求第一个 DNS Server，如果1秒内获得解析结果，则返回结果；
- 第0秒请求第一个 DNS Server，如果超过1秒还未获得解析结果，则在1秒后开始请求第2个DNS Server，并返回最快获得的解析结果；
- 以此类推；

### DNS记录配置

自定义的DNS记录配置只需在命令行参数`-conf_dir`指定的配置目录中添加以`.dns-conf`后缀结尾的文件即可。可以分多个文件，也可以是在子目录里面，只要是以`.dns-conf`后缀结尾就行。    

可以根据不同需求组织目录和文件，例如：

```
conf
├── k8s
│   └── qa-k8s.dns-conf
├── mydomain.com.dns-conf
├── resolv.conf
└── test.dns-conf
```

### 解析顺序

fpdns解析dns请求的时候，会按照以下逻辑进行处理：

```
1. .dns-conf 配置文件是否配置了对应记录
	2. 是：
		3. 返回配置的记录
	4. 否：
		5. 检查缓存中是否有对应的缓存记录
			6. 是：
				7. 检查缓存记录是否过期
					8. 已过期，跳到 11
					9. 未过期，返回缓存的记录值
			10. 否：
				11. 查询 resolv.conf 配置的上游DNS服务器
```

### 自定义A记录配置

A记录配置格式为：

```
域名. TTL(Time-To-Live，单位秒) IN A 目标IP地址
```

A记录配置参考以下示例：

```
about.fpdns.cn. 600  IN  A  192.168.2.19
```

### 自定义A记录负载均衡

同一个域名添加多个A记录的时候，就会开启DNS负载均衡。

例如以下配置：

```
about.fpdns.cn. 600  IN  A  192.168.2.19
about.fpdns.cn. 600  IN  A  192.168.2.20
about.fpdns.cn. 600  IN  A  192.168.2.21
```

因不少应用会直接拿DNS解析结果的第一个IP来使用，所以fpdns每次解析的时候，都会随机乱序返回。

### 泛解析

支持A记录泛解析，例如：

```
*.github.com. 600 IN A 192.168.1.253
```

以上的配置会对 `github.com` 的所有子域名（包括多级子域名）都生效，例如 `a.github.com`, `b.a.github.com`, `c.b.a.github.com`，但不包括 `github.com`。

### 自定义CNAME记录配置

CNAME记录配置格式为：

```
域名. TTL(Time-To-Live，单位秒) IN CNAME 目标域名
```

CNAME记录配置参考以下示例：

```
www.baidu.com. 172800  IN  CNAME  www.a.shifen.com
```

### DIRECT记录

当需要某个域名直接去查询 `resolv.conf` 里面的上游DNS服务的时候，可以配置为 `CNAME DIRECT`，则该记录会查询上游DNS服务器。

例如以下配置：

```
*.test1.com. 172800  IN      A       192.168.2.22
up.test1.com. 172800  IN      CNAME       DIRECT
```

以上配置 `*.test1.com` 会将所有子域名都解析到 `192.168.2.22` ，但是 `up.test1.com` 会使用上游DNS服务器进行解析，而不是解析到 `192.168.2.22`。


### 自定义DNS反向查询

DNS反向查询PTR，就是例如`dig -x 8.8.8.8 +short`返回`google-public-dns-a.google.com.`，通过IP查询对应的域名。

示例配置文件：

```
# 注意这里是 PTR记录 而不是 A记录
192.168.3.253. 172800  IN  PTR   c253.fpdns.cn
192.168.3.254. 172800  IN  PTR   c254.fpdns.cn

about.fpdns.cn. 600  IN  A  192.168.2.19

```

上面的配置文件中`PTR`类型的 `c253.fpdns.cn.` 和 `c254.fpdns.cn.` 可以支持DNS反向查询，就是`dig -x 192.168.3.253 +short` 会返回 `c253.fpdns.cn.` 。

## HTTP接口

### /debug 接口

打印一些调试信息。

```
curl "http://host:port/debug"
```

响应内容：

```
Local config cache len:
	[class:IN, type:A]: 1028

Resolv cache len: 2656

DNS Query QPS: 101.200000
```

### /reload_conf 接口

修改 `*.dns-conf` 配置文件后，调用这个接口可以`重新加载配置`，而不需要重启服务。


```
curl "http://host:port/reload_conf"
```

响应内容：

```
reload dns conf done.

dns conf add: 4
	 add new1new.web.: class:IN, type:A
	 add new-new.web.: class:IN, type:A
	 add c252.fpdns.cn.: class:IN, type:A
	 add c254.fpdns.cn.: class:IN, type:A

dns conf delete: 2
	 delete newnew.web.: class:IN, type:A
	 delete new2new.web.: class:IN, type:A

dns conf change: 2
	 change about.fpdns.com.: class:IN, type:A
	 change hello.fpdns.com.: class:IN, type:A
```
