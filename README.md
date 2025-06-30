# Discord Bot Luna 🌙

多機能を目指して開発中のGo言語製Discordボットです。

[![Go Version](https://img.shields.io/badge/Go-1.18%2B-blue.svg)](https://golang.org/)
[![DiscordGo](https://img.shields.io/badge/lib-DiscordGo-blue.svg)](https://github.com/bwmarrin/discordgo)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## ✨ 機能一覧

Lunaは、サーバー管理を効率化し、コミュニティを活性化させるための様々な機能を搭載しています。

* ** SLASHコマンド**: 直感的で分かりやすいスラッシュコマンドに対応。
* **⚙️ ユーティリティ**: サーバーやユーザーの情報を簡単に確認できます。
* **🎫 チケットシステム**: ボタン一つで、ユーザーごとのプライベートなサポートチャンネルを作成・管理できます。
* **📝 モジュール化**: コマンドごとにファイルが整理されており、新しい機能の追加やメンテナンスが容易な設計です。

---

## 🚀 導入方法

### 1. 前提条件
* [Go言語](https://go.dev/dl/) (バージョン 1.18以上) がインストールされていること。

### 2. 設定
1.  このリポジトリをクローンまたはダウンロードします。
    ```bash
    git clone [https://github.com/pepeyukke/luna.git](https://github.com/pepeyukke/luna.git)
    cd luna
    ```
2.  必要なライブラリをインストールします。
    ```bash
    go mod tidy
    ```
3.  [Discord Developer Portal](https://discord.com/developers/applications) で取得した**ボットトークン**を環境変数に設定します。

    **Windows (PowerShell)**
    ```powershell
    $env:DISCORD_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"
    ```
    **macOS / Linux**
    ```bash
    export DISCORD_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"
    ```

### 3. 実行
以下のコマンドでボットを起動します。
```bash
go run .
```

### 4. サーバーへの招待
[Discord Developer Portal](https://discord.com/developers/applications)の`OAuth2 > URL Generator`から、以下のスコープと権限を選択して招待リンクを生成してください。
* **SCOPES**: `bot`, `applications.commands`
* **BOT PERMISSIONS**:
    * `Send Messages`
    * `Embed Links`
    * `Read Messages/View Channels`
    * `Manage Channels` (チケット機能で必要)

---

## 📋 コマンドリスト

| コマンド | 説明 | 使い方 |
|:---|:---|:---|
| `/ping` | ボットの応答速度(レイテンシ)を測定します。 | `/ping` |
| `/avatar` | あなた、または指定したユーザーのアバターを表示します。| `/avatar user:@ユーザー名` |
| `/embed` | 指定した内容でEmbedメッセージを自由に作成します。 | `/embed title:タイトル description:説明 ...` |
| `/ticket-setup` | サポートチケット作成用のパネルを設置します。(管理者用) | `/ticket-setup channel:#チャンネル staff-role:@ロール`|

---

## ライセンス

このプロジェクトは [LGPL-3.0](LICENSE) の下で公開されています。