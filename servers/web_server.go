package servers

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec" // os/execをインポート
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
	ws := &WebServer{
		log: log,
		db:  db,
	}

	r := mux.NewRouter()
	authHandler := web.NewAuthHandler(log)

	r.HandleFunc("/api/auth/login", authHandler.Login).Methods("GET")
	r.HandleFunc("/api/auth/callback", authHandler.Callback).Methods("GET")
	r.HandleFunc("/api/auth/logout", authHandler.Logout).Methods("GET")
	r.HandleFunc("/api/auth/user", authHandler.GetUser).Methods("GET")
	r.HandleFunc("/api/dashboard", ws.DashboardHandler).Methods("GET")

	ws.http = &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	return ws
}

// Start はWebサーバーを起動します。
func (s *WebServer) Start() error {
	s.log.Info("Webサーバーを http://localhost:8080 で起動します")
	return s.http.ListenAndServe()
}

// Stop はWebサーバーをシャットダウンします。
func (s *WebServer) Stop() error {
	s.log.Info("Webサーバーをシャットダウンします...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.http.Shutdown(ctx); err != nil {
		s.log.Error("Webサーバーのシャットダウンに失敗しました", "error", err)
		return err
	}
	return nil
}

// Name はサーバーの名前を返します。
func (s *WebServer) Name() string {
	return "Web Server"
}

// Cmd はサーバーの実行コマンドを返します。Webサーバーの場合はnilを返します。
func (s *WebServer) Cmd() *exec.Cmd {
	return nil
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
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.log.Error("Failed to encode dashboard data", "error", err)
	}
}