
// servers/web_server.go
package servers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"luna/handlers/web"
	"luna/interfaces"

	"github.com/gorilla/mux"
)

// WebServer はHTTPサーバーを管理します。
type WebServer struct {
	log    interfaces.Logger
	db     interfaces.DataStore
	http   *http.Server
}

// NewWebServer は新しいWebServerインスタンスを作成します。
func NewWebServer(log interfaces.Logger, db interfaces.DataStore) *WebServer {
	r := mux.NewRouter()

	// APIハンドラのインスタンスを作成
	authHandler := web.NewAuthHandler(log)

	// ルーティングを設定
	r.HandleFunc("/api/auth/login", authHandler.Login).Methods("GET")
	r.HandleFunc("/api/auth/callback", authHandler.Callback).Methods("GET")

	r.HandleFunc("/api/dashboard", s.DashboardHandler).Methods("GET")

	// TODO: 他のAPIエンドポイントをここに追加

	return &WebServer{
		log: log,
		db:  db,
		http: &http.Server{
			Addr:    ":8080", // ポートは後で設定ファイルから読み込むように変更
			Handler: r,
		},
	}
}

// Start はWebサーバーを起動します。
func (s *WebServer) Start() error {
	s.log.Info("Webサーバーを http://localhost:8080 で起動します")
	return s.http.ListenAndServe()
}

// Stop はWebサーバーをシャットダウンします。
func (s *WebServer) Stop() {
	// ... (Stop method implementation)
}

// DashboardHandler はダッシュボードのサマリーデータを返します。
func (s *WebServer) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// TODO: 実際のデータをデータベースから取得する
	data := map[string]interface{}{
		"totalUsers":      1234,
		"onlineUsers":     567,
		"totalServers":    12,
		"commandsExecuted": 8901,
		"commandUsage": []map[string]interface{}{
			{"name": "/ask", "count": 4000},
			{"name": "/imagine", "count": 3000},
			{"name": "/quiz", "count": 2000},
			{"name": "/poll", "count": 2780},
			{"name": "/ping", "count": 1890},
			{"name": "/help", "count": 2390},
			{"name": "/avatar", "count": 3490},
		},
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.log.Error("Failed to encode dashboard data", "error", err)
	}
}

	s.log.Info("Webサーバーをシャットダウンします...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.http.Shutdown(ctx); err != nil {
		s.log.Error("Webサーバーのシャットダウンに失敗しました", "error", err)
	}
}
