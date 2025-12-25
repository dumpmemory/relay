package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/DGHeroin/relay/webui/model"
)

// 版本信息，通过 ldflags 注入
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

var (
	addr        = flag.String("addr", ":8080", "监听地址")
	showVersion = flag.Bool("version", false, "显示版本信息")
	dataDir     = "data"
)

func main() {
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		log.Printf("Relay WebUI %s\n", Version)
		log.Printf("Build Time: %s\n", BuildTime)
		log.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Relay WebUI %s starting...", Version)

	// 获取可执行文件目录
	execPath, err := os.Executable()
	if err == nil {
		dataDir = filepath.Join(filepath.Dir(execPath), "data")
	}

	// 初始化数据库
	if err := model.InitDB(dataDir); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer model.CloseDB()

	// 创建服务器
	server := NewServer(*addr)

	// 自动启动已启用的规则
	if model.IsSetupCompleted() {
		autoStart, _ := model.GetSetting("auto_start")
		if autoStart == "true" {
			go func() {
				rules, err := model.GetEnabledRelayRules()
				if err != nil {
					log.Printf("获取规则失败: %v", err)
					return
				}
				for _, rule := range rules {
					if err := server.handlers.relayMgr.Start(rule, server.handlers.wsHub, server.handlers.geoIP); err != nil {
						log.Printf("自动启动失败 %s: %v", rule.Name, err)
					}
				}
			}()
		}

		// 加载 GeoIP
		geoEnabled, _ := model.GetSetting("geoip_enabled")
		if geoEnabled == "true" {
			geoPath := filepath.Join(dataDir, "GeoLite2-City.mmdb")
			if err := server.handlers.geoIP.Load(geoPath); err != nil {
				log.Printf("GeoIP 加载失败: %v", err)
			}
		}
	}

	// 启动定时清理 (每天执行一次)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		// 启动时先清理一次
		model.CleanOldStats()
		for range ticker.C {
			model.CleanOldStats()
		}
	}()

	// 优雅退出
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("正在关闭...")
		server.handlers.relayMgr.StopAll()
		model.CloseDB()
		os.Exit(0)
	}()

	// 启动服务器
	if err := server.Run(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
