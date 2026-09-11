package main

// ws_gateway.go — 统一前端入口 :48092（浏览器式导航外壳 + API 网关）。
//
// 此前 README 宣告 :48092 网关但仓库里没有监听实现（selfheal 探活项一直显示 unhealthy），
// 本文件把宣称补成现实：
//   /            → uiteg（worldapp 构建产物：浏览器式导航外壳，SPA fallback 到 index.html）
//   /api/novel/* → :48090/api/*（小说创作服务，show-me-the-story 流水线）
//   /api/*       → :48091/api/*（世界模拟服务，含文字游戏 /api/game/*）
//   /game        → :48091/game（终端风文字游戏页）
// 其余静态路径按文件名查 uiteg；未命中回退 index.html（客户端路由）。

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func startUnifiedEntry() {
	mux := http.NewServeMux()

	// 小说 API：网关 /api/novel/X → 48090 /api/X
	novelTarget, _ := url.Parse("http://127.0.0.1" + storyPort)
	mux.Handle("/api/novel/", &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = novelTarget.Scheme
			req.URL.Host = novelTarget.Host
			req.URL.Path = "/api" + strings.TrimPrefix(req.URL.Path, "/api/novel")
			req.Host = req.URL.Host
		},
	})
	// 世界 API（含 /api/game/*）与 /game 页 → 48091
	worldTarget, _ := url.Parse("http://127.0.0.1" + worldPort)
	mux.Handle("/api/", &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = worldTarget.Scheme
			req.URL.Host = worldTarget.Host
			req.Host = req.URL.Host
		},
	})
	mux.Handle("/game", &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = worldTarget.Scheme
			req.URL.Host = worldTarget.Host
			req.Host = req.URL.Host
		},
	})

	// uiteg 静态外壳（SPA fallback）
	sub, err := fs.Sub(uitegFS, "uiteg")
	if err != nil {
		return
	}
	shell := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, openErr := sub.Open(path); openErr == nil {
			_ = f.Close()
			shell.ServeHTTP(w, r)
			return
		}
		// SPA fallback：客户端路由接管未知路径
		data, _ := fs.ReadFile(sub, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})

	if err := http.ListenAndServe(listenHost()+uiPort, mux); err != nil {
		// 网关失败不崩主进程：世界/小说服务仍可用，只在 stdout 提示
		println(" [警告] 统一前端入口启动失败: " + err.Error())
	}
}
