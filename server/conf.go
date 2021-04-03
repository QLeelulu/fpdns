package server

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/miekg/dns"
)

func loadConf(_confDir string) {
	err := filepath.Walk(_confDir, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return nil
		}
		if f.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".dns-conf") {
			loadDNSConf(path, f, nil)
		} else if filepath.Base(path) == "resolv.conf" {
			resolvConfFile = path
		}
		return nil
	})
	if err != nil {
		// fmt.Printf("filepath.Walk() returned %v\n", err)
		logInstance.Errorf("filepath.Walk() returned: %v\n", err)
	}
}

func loadDNSConf(path string, f os.FileInfo, reloadRRCache map[string]map[[2]uint16][]dns.RR) {
	rrCache2 := rrCache
	if reloadRRCache != nil {
		rrCache2 = reloadRRCache
	}

	inFile, err := os.Open(path)
	if err != nil {
		logInstance.Errorf("load conf file [%s] error: %s \n", path, err)
		return
	}
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		rrConf := strings.TrimSpace(scanner.Text())
		if len(rrConf) < 1 || strings.HasPrefix(rrConf, "#") {
			continue
		}
		rr, err := dns.NewRR(rrConf)
		if err != nil {
			logInstance.Warnf("load conf [%s] from file [%s] error: %s\n", rrConf, path, err)
		} else {
			h := rr.Header()
			if h == nil {
				logInstance.Warnf("load conf [%s] from file [%s] error: can't get RR_Header\n", rrConf, path)
			} else {
				if h.Rrtype == dns.TypePTR { // PTR 反向解析需要特殊处理一下
					h.Name, err = dns.ReverseAddr(strings.Trim(h.Name, "."))
					if err != nil {
						logInstance.Warnf("load conf [%s] from file [%s] error: wrong PRT record, %s\n", rrConf, path, err)
						continue
					}
				}
				h.Name = strings.ToLower(h.Name)
				// c := rrCache2[[2]uint16{h.Class, h.Rrtype}]
				c := rrCache2[h.Name]
				if c == nil {
					c = map[[2]uint16][]dns.RR{}
					rrCache2[h.Name] = c
				}
				c[[2]uint16{h.Class, h.Rrtype}] = append(c[[2]uint16{h.Class, h.Rrtype}], rr)
			}
		}
	}
}

// reloadDNSConf 重新加载dns配置，并且返回增加、删除和修改过的记录
func reloadDNSConf() (add, del, change []string) {
	newRRCache := map[string]map[[2]uint16][]dns.RR{}
	// newRRCache[[2]uint16{dns.ClassINET, dns.TypeA}] = map[string][]dns.RR{}

	err := filepath.Walk(sc.ConfDir, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return nil
		}
		if f.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".dns-conf") {
			loadDNSConf(path, f, newRRCache)
		}
		return nil
	})
	if err != nil {
		// fmt.Printf("filepath.Walk() returned %v\n", err)
		logInstance.Errorf("filepath.Walk() returned: %v\n", err)
	}

	add, del, change = getDiffDNSConf(rrCache, newRRCache)
	rrCache = newRRCache
	return
}

// getDiffDNSConf 返回增加、删除和修改过的记录
func getDiffDNSConf(oldC, newC map[string]map[[2]uint16][]dns.RR) (add, del, change []string) {
	for name, newM := range newC {
		if oldM, ok := oldC[name]; ok {
			for t, newRR := range newM {
				if oldRR, ok := oldM[t]; ok {
					sNewRR := make([]string, 0, len(newRR))
					sOldRR := make([]string, 0, len(oldRR))
					for _, v := range newRR {
						sNewRR = append(sNewRR, v.String())
					}
					for _, v := range oldRR {
						sOldRR = append(sOldRR, v.String())
					}
					if !reflect.DeepEqual(sNewRR, sOldRR) {
						for _, rr := range newRR {
							change = append(change, fmt.Sprintf("%s", rr.String()))
						}
					}
				} else {
					for _, rr := range newRR {
						add = append(add, fmt.Sprintf("%s", rr.String()))
					}
				}
			}
		} else {
			for _, newRR := range newM {
				for _, rr := range newRR {
					add = append(add, fmt.Sprintf("%s", rr.String()))
				}

			}
		}
	}
	for name, oldRR := range oldC {
		if _, ok := newC[name]; !ok {
			for _, rrs := range oldRR {
				for _, rr := range rrs {
					del = append(del, fmt.Sprintf("%s", rr.String()))
				}
			}
		}
	}
	return
}
