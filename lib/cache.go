package lib

import (
	"time"

	"github.com/allegro/bigcache"
	"github.com/miekg/dns"
)

type KeyNotFound struct {
}

func (e KeyNotFound) Error() string {
	return "not found"
}

type KeyExpired struct {
}

func (e KeyExpired) Error() string {
	return "expired"
}

var (
	KeyNotFoundError = KeyNotFound{}
	KeyExpiredError  = KeyExpired{}
)

type Mesg struct {
	Msg    *dns.Msg
	Expire time.Time
}

type MemoryCache struct {
	// bigcache过期是不会删数据的，只有当存储的数据
	// 多于我们设定的范围，才会用新值覆盖旧值，所以需
	// 要额外处理过期时间的问题。
	cache  *bigcache.BigCache
	Expire time.Duration
}

func NewMemoryCache(ttl int) (*MemoryCache, error) {
	config := bigcache.Config{
		// number of shards (must be a power of 2)
		Shards: 128,
		// time after which entry can be evicted
		LifeWindow: time.Duration(ttl) * time.Second,
		// rps * lifeWindow, used only in initial memory allocation
		MaxEntriesInWindow: 1000 * 10 * 60,
		// max entry size in bytes, used only in initial memory allocation
		MaxEntrySize: 300,
		// prints information about additional memory allocation
		Verbose: false,
		// cache will not allocate more memory than this limit, value in MB
		// if value is reached then the oldest entries can be overridden for the new ones
		// 0 value means no size limit
		HardMaxCacheSize: 2048,
		// callback fired when the oldest entry is removed because of its
		// expiration time or no space left for the new entry. Default value is nil which
		// means no callback and it prevents from unwrapping the oldest entry.
		OnRemove: nil,
	}

	cache, initErr := bigcache.NewBigCache(config)
	if initErr != nil {
		return nil, initErr
	}
	mc := &MemoryCache{}
	mc.cache = cache
	mc.Expire = config.LifeWindow
	return mc, nil
}

func (c *MemoryCache) Get(q dns.Question) (*dns.Msg, error) {
	key := q.String()
	v, err := c.cache.Get(key)
	if err != nil {
		// fmt.Println(err)
		switch err {
		case bigcache.ErrEntryNotFound:
			// 不处理
		default:
			AppLog().Errorln("bigcache get error: ", err)
		}
		return nil, KeyNotFoundError
	}
	// 前15个为过期时间
	if len(v) < 16 {
		AppLog().Warnln("cache value's len less than 15")
		return nil, KeyNotFoundError
	}

	expireb := v[:15]
	var expire time.Time
	err = expire.UnmarshalBinary(expireb)
	if err != nil {
		AppLog().Errorln("UnmarshalBinary error:", err)
		return nil, KeyNotFoundError
	}

	var msg dns.Msg
	err = msg.Unpack(v[15:])
	if err != nil {
		AppLog().Errorln("msg.Unpack error: ", err)
		return nil, KeyNotFoundError
	}

	if expire.Before(time.Now()) {
		return &msg, KeyExpiredError
	}

	return &msg, nil

}

func (c *MemoryCache) Set(q dns.Question, msg *dns.Msg) error {
	expire := time.Now().Add(c.Expire)
	expireb, err := expire.MarshalBinary()
	if err != nil {
		return err
	}

	v, err := msg.Pack()
	if err != nil {
		return err
	}

	v = append(expireb, v...)
	// fmt.Println("val len: ", len(v))

	key := q.String()
	// fmt.Println("set", key)

	return c.cache.Set(key, v)
}

func (c *MemoryCache) Length() int {
	return c.cache.Len()
}
