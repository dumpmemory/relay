package service

import (
	"net"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

// GeoIPService GeoIP 服务
type GeoIPService struct {
	db   *maxminddb.Reader
	mu   sync.RWMutex
	path string
}

// geoRecord GeoIP 记录
type geoRecord struct {
	Country struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
}

// NewGeoIPService 创建服务
func NewGeoIPService() *GeoIPService {
	return &GeoIPService{}
}

// Load 加载数据库
func (g *GeoIPService) Load(path string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 关闭旧的
	if g.db != nil {
		g.db.Close()
		g.db = nil
	}

	db, err := maxminddb.Open(path)
	if err != nil {
		return err
	}

	g.db = db
	g.path = path
	return nil
}

// Close 关闭数据库
func (g *GeoIPService) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.db != nil {
		g.db.Close()
		g.db = nil
	}
	g.path = ""
}

// IsLoaded 是否已加载
func (g *GeoIPService) IsLoaded() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.db != nil
}

// Lookup 查询 IP 地理位置
func (g *GeoIPService) Lookup(ipStr string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.db == nil {
		return ""
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}

	var record geoRecord
	if err := g.db.Lookup(ip, &record); err != nil {
		return ""
	}

	country := record.Country.Names["zh-CN"]
	if country == "" {
		country = record.Country.Names["en"]
	}

	city := record.City.Names["zh-CN"]
	if city == "" {
		city = record.City.Names["en"]
	}

	if country == "" {
		return ""
	}

	if city != "" && city != country {
		return country + "/" + city
	}
	return country
}
