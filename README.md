# Luna - Version 2.0.8

Lunaは、Goの高性能な多機能Botです。将来的にはReact製のWebダッシュボードからの操作も可能です。

## ✨ 主要機能

Lunaは、あなたのDiscordサーバーをより楽しく、より便利にするための多彩な機能を提供します。

- **AIアシスタント機能 (Luna Assistant)**
  - `/ask`: Lunaに質問を投げかけると、文脈を理解して的確に回答します。
  - `/imagine`: 最新の画像生成AI（Luna Assitant Imagen Module）で、あなたのアイデアを美しい画像に変換します。
    - ネガティブプロンプトや、AIによる自動画質向上機能のON/OFFもサポート。
  - `/describe_image`: 画像をアップロードまたはURLを指定すると、その内容を詳細に説明します。

- **高音質な音楽再生**
  - `/play`: YouTube動画のURLや検索クエリから音楽を再生します。
  - `/queue`: 再生待ちの曲リストを表示・管理します。
  - `/skip`, `/stop`, `/leave`: 音楽再生を直感的にコントロール。

- **サーバー管理・便利ツール**
  - `/poll`: サーバー内で簡単に投票を作成できます。
  - `/moderate`: メッセージの削除など、モデレーション作業を支援します。
  - `/user_info`: ユーザーの情報を表示します。
  - `/ticket`: サポートや問い合わせのためのチケットを発行・管理します。
  - `/word_ranking`: サーバー内での単語使用頻度をランキング表示します。

- **エンターテイメント**
  - `/quiz`: 様々なトピックの4択クイズを出題します。
  - `/roulette`: ランダムな選択やゲームに使えます。

## 💻 技術スタック

Lunaの技術スタック情報。

- **バックエンド:** Go
- **AIサーバー:** Python (Flask) + Google Vertex AI (Gemini, Imagen)
- **フロントエンド:** React, TypeScript (Webダッシュボード)
- **データベース:** SQLite
- **ライブラリ:**
  - `discordgo` (Go)
  - `viper` (Go)
  - `gocron` (Go)
  - `vertexai` (Python)
  - `flask` (Python)

## 🚀セットアップ
### 1. 前提

- Go (1.21以上)
- Python (3.13以上)
- Node.js (18.x以上)

### 2. クローン

```bash
git clone https://github.com/your-username/luna.git
cd luna
```

### 3. 設定ファイル

プロジェクトのルートに `config.yaml` ファイルを作成し、以下の内容を記述してください。

```yaml
discord:
  token: "YOUR_DISCORD_BOT_TOKEN_HERE"

google:
  project_id: "YOUR_GOOGLE_CLOUD_PROJECT_ID"
  credentials_path: "path/to/your/google_credentials.json" # 省略可

web:
  client_id: "YOUR_DISCORD_OAUTH_CLIENT_ID"
  client_secret: "YOUR_DISCORD_OAUTH_CLIENT_SECRET"
  redirect_uri: "http://localhost:3000/auth/callback"
  session_secret: "a-very-secret-key-for-sessions"
```

- **Discord Token:** [Discord Developer Portal](https://discord.com/developers/applications) でBotを作成し、トークンを取得します。`Bot` と `applications.commands` のスコープ権限、そして必要な特権インテント（サーバーメンバー、メッセージ内容）を有効にしてください。
- **Google Cloud:** Vertex AIを使用するために、Google CloudプロジェクトのIDと、サービスアカウントの認証情報（JSONファイル）へのパスを設定します。
- **Web OAuth:** Webダッシュボード機能のためのDiscord OAuth2設定です。

### 4. AIサーバー

```bash
cd python_server
pip install -r requirements.txt
```

### 5. フロントエンド

```bash
cd frontend/frontend
npm install
npm start
```

### 6. 起動

```bash
go mod tidy
go run main.go
```

## 使い方

1.  [Discord Developer Portal](https://discord.com/developers/applications) のあなたのBotのページで、`OAuth2 > URL Generator` を開きます。
2.  `bot` と `applications.commands` のスコープを選択します。
3.  必要なBot権限（管理者権限を推奨）を選択し、生成されたURLを使ってあなたのDiscordサーバーにBotを招待します。
4.  サーバー内で `/` を入力すると、利用可能なコマンドの一覧が表示されます。

## 📜 ライセンス

このプロジェクトは [LGPL-3.0](LICENSE.md) の下で公開されています。
