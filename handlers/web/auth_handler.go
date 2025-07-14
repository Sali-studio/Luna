package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"luna/config"
	"luna/interfaces"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

const (
	discordAPIEndpoint = "https://discord.com/api/v10"
	oauthStateString   = "random-string-for-security"
)

var (
	store       *sessions.CookieStore
	oauth2Config *oauth2.Config
)

// InitAuth は認証ハンドラを初期化します。
func InitAuth(cfg *config.Config) {
	store = sessions.NewCookieStore([]byte(cfg.Web.SessionSecret))
	oauth2Config = &oauth2.Config{
		ClientID:     cfg.Web.ClientID,
		ClientSecret: cfg.Web.ClientSecret,
		RedirectURL:  cfg.Web.RedirectURI,
		Scopes:       []string{"identify", "guilds"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  discordAPIEndpoint + "/oauth2/authorize",
			TokenURL: discordAPIEndpoint + "/oauth2/token",
		},
	}
}

type AuthHandler struct {
	log interfaces.Logger
}

func NewAuthHandler(log interfaces.Logger) *AuthHandler {
	return &AuthHandler{log: log}
}

// Login はユーザーをDiscordの認証ページにリダイレクトします。
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	rand.Read(b)
	oauthState := base64.URLEncoding.EncodeToString(b)

	session, _ := store.Get(r, "session-name")
	session.Values["oauth_state"] = oauthState
	session.Save(r, w)

	http.Redirect(w, r, oauth2Config.AuthCodeURL(oauthState), http.StatusTemporaryRedirect)
}

// Callback はDiscordからのコールバックを処理します。
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if r.URL.Query().Get("state") != session.Values["oauth_state"] {
		http.Error(w, "Invalid session state", http.StatusUnauthorized)
		return
	}

	token, err := oauth2Config.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange token: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// ユーザー情報を取得
	res, err := oauth2Config.Client(context.Background(), token).Get(discordAPIEndpoint + "/users/@me")
	if err != nil || res.StatusCode != 200 {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	// ユーザー情報をセッションに保存
	session.Values["access_token"] = token.AccessToken
	session.Values["user_info"] = string(body)
	session.Save(r, w)

	// フロントエンドのダッシュボードにリダイレクト
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// Logout はユーザーセッションを破棄します。
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Options.MaxAge = -1
	session.Save(r, w)
	w.WriteHeader(http.StatusOK)
}

// GetUser は現在のユーザー情報を返します。
func (h *AuthHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	userInfo, ok := session.Values["user_info"].(string)

	if !ok || userInfo == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var user map[string]interface{}
	json.Unmarshal([]byte(userInfo), &user)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}