# Luna - 2.0.11

Lunaは、AI、カジノゲーム、音楽再生、サーバー管理など、多彩な機能を備えた多機能Discordボットです。

## ✨ 主な機能

- **AI機能 (Luna AI):**
  - `/ask`: AIに質問できます。
  - `/imagine`: 指示に基づいて画像を生成します。
  - `/describe-image`: 画像の内容を説明します。
  - `/ocr`: 画像からテキストを抽出します。
  - `/quiz`: AIが生成したクイズに挑戦できます。
  - `/profile`: ユーザーの活動履歴からAIが分析します。

- **カジノ & ゲーム機能:**
  - `/daily`: 毎日チップを2000受け取れます。
  - `/balance`: チップの残高を確認します。
  - `/leaderboard`: チップの所持数ランキングを表示します。
  - `/pay`: 他のユーザーにチップを送金します。
  - `/slots`: スロットマシンをプレイします。
  - `/coinflip`: コイントスでギャンブルします。
  - `/horserace`: 競馬にベットしてレースを観戦します。
  - `/quizbet`: AIクイズにチップを賭けて挑戦します。
  - `/blackjack`: ディーラーとブラックジャックで勝負します。

- **音楽再生機能(破損):**
  - `/join`: ボイスチャンネルに参加します。
  - `/play`: YouTubeの動画やプレイリストを再生します。
  - `/stop`: 再生を停止します。
  - `/skip`: 現在の曲をスキップします。
  - `/queue`: 再生キューを表示します。
  - `/leave`: ボイスチャンネルから退出します。

- **サーバー管理 & ユーティリティ:**
  - `/config`: サーバー固有の設定を管理します。
  - `/ticket`: サポート用のチケットを作成します。
  - `/poll`: 投票を作成します。
  - `/moderate`: メッセージの削除など、モデレーションを行います。
  - その他、アバター表示、電卓、翻訳など多数の便利コマンド。

- **Webダッシュボード (開発中):**
  - サーバーの設定や統計情報をWebブラウザから確認できます。

## 🛠️ アーキテクチャ

Lunaは、Go言語で書かれたメインのボットアプリケーションと、AI機能を提供するためのPythonサーバーで構成されています。

- **バックエンド (Go):**
  - `discordgo` ライブラリを使用してDiscord APIと通信します。
  - コマンド処理、イベントハンドリング、データベースとの連携を担当します。
  - データベースには `SQLite` を使用しており、ユーザーデータやサーバー設定を永続化します。

- **AIサーバー (Python):**
  - `Flask` フレームワークで構築されたAPIサーバーです。
  - Googleの `Vertex AI` (Gemini) と連携し、テキスト生成、画像生成、画像認識などの高度なAI機能を提供します。
  - Goバックエンドからのリクエストに応じて、AIモデルを呼び出します。

## 🚀 セットアップ方法

### 必要なもの

- Go (1.24.4 以上)
- Python (3.x)
- Google Cloud Platform (GCP) アカウントとプロジェクト
  - Vertex AI APIが有効になっていること
  - GCPの認証情報 (サービスアカウントキー) JSONファイル

### 1. リポジトリのクローン

```bash
git clone https://github.com/your-username/luna.git
cd luna
```

### 2. Goバックエンドの設定

1.  Goの依存関係をインストールします。

    ```bash
    go mod tidy
    ```

2.  設定ファイル `config.yaml` をプロジェクトのルートディレクトリに作成します。`config.example.yaml` を参考に、以下の内容を記述してください。

    ```yaml
    discord:
      token: "YOUR_DISCORD_BOT_TOKEN"

    google:
      project_id: "YOUR_GCP_PROJECT_ID"
      credentials_path: "path/to/your/gcp-credentials.json"

    web:
      client_id: "YOUR_DISCORD_APP_CLIENT_ID"
      client_secret: "YOUR_DISCORD_APP_CLIENT_SECRET"
      redirect_uri: "http://localhost:8080/auth/callback"
      session_secret: "a-very-secret-key-for-sessions"
    ```

### 3. Python AIサーバーの設定

1.  Pythonの依存関係をインストールします。

    ```bash
    pip install -r requirements.txt
    ```

2.  GCPの認証情報が正しく設定されていることを確認してください。(`config.yaml` の `credentials_path`)

### 4. 起動

メインのGoアプリケーションを起動すると、PythonのAIサーバーも自動的に起動します。

```bash
go run main.go
```

## 🤝 貢献

バグ報告や機能提案は、GitHubのIssuesまでお気軽にどうぞ。