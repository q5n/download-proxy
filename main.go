package main

import (
	"log"
	"net/http"
	"time"

	"github.com/q5n/download-proxy/internal/config"
	"github.com/q5n/download-proxy/internal/proxy"
)

// main starts the download proxy HTTP server. / 主函数：启动下载代理 HTTP 服务。
func main() {
	// Load configuration and initialize the proxy server. / 加载配置并初始化代理服务。
	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(err)
	}

	p := proxy.New(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/download", p.Handler)

	srv := &http.Server{
		Addr:         cfg.Listen,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("listen", cfg.Listen)

	err = srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
