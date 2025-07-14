// handlers/web/auth_handler.go
package web

import (
	"fmt"

	"luna/interfaces"
	"net/http"
)

type AuthHandler struct {
	log interfaces.Logger
}

func NewAuthHandler(log interfaces.Logger) *AuthHandler {
	return &AuthHandler{log: log}
}

// Login はDiscord OAuth2のログインフローを開始します。
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// TODO: Discord OAuth2のURLにリダイレクトするロジックを実装
	// 現時点では、プレースホルダーメッセージを返す
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "This is the login endpoint. It will redirect to Discord.")
}

// Callback はDiscordからの認証コールバックを処理します。
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	// TODO: Discordから受け取ったコードを使ってアクセストークンを取得し、
	// ユーザー情報を取得してセッションを開始するロジックを実装
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "This is the callback endpoint. User is now authenticated.")
}
