# Luna 1.6.3

Go言語で開発された、サーバー管理、AIとの対話、そして多彩なユーティリティ機能を提供する、次世代の多機能Discordボットです。

[![Go Version](https://img.shields.io/badge/Go-1.18%2B-blue.svg)](https://golang.org/)
[![DiscordGo](https://img.shields.io/badge/lib-DiscordGo-blue.svg)](https://github.com/bwmarrin/discordgo)
[![License](https://img.shields.io/badge/License-LGPL--3.0-blue.svg)](LICENSE)

---

## ✨ 機能一覧

Lunaは、サーバー運営を円滑にし、コミュニティを活性化させるための幅広い機能を、洗練されたUI/UXで提供します。

* **🤖 AIチャット**: GoogleのGemini APIと連携し、ユーザーの質問にAIが自然に応答します。
* **🎫 高度なチケットシステム**: ボタンとモーダル形式で問い合わせを受け付け、AIによる一次回答を提示することで、サポート業務を効率化します。
* **🛡️ モデレーション**: Kick, BAN, Timeoutなどの基本的な管理機能を、理由付きで素早く実行できます。
* **📈 詳細なロギング**: サーバー内のあらゆる重要イベントを、見やすいEmbed形式でリアルタイムに記録し、監査ログを強力に補完します。
* **🔧 便利なツール群**:
  - **高機能電卓**: 括弧や関数にも対応した、強力な計算機機能。
  - **ポケモン実数値計算機**: レベル、個体値、努力値から性格・ランク・アイテム補正まで、対戦環境に即した詳細なステータス計算が可能。
  - **工業MOD電力変換機**: Minecraftの工業MODプレイヤー向けに、各種電力単位を相互変換します。
* **⚙️ ユーティリティ**: ユーザーアバターの表示や、カスタムEmbedの作成など、コミュニケーションを豊かにする機能を提供します。

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
    go get [github.com/Knetic/govaluate](https://github.com/Knetic/govaluate)
    ```
3.  各種**APIキー**を環境変数に設定します。

    **Windows (PowerShell)**
    ```powershell
    # Discordボットのトークン
    $env:DISCORD_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"

    # Google AI Studioで取得したGemini APIキー
    $env:GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
    ```
    **macOS / Linux**
    ```bash
    export DISCORD_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"
    export GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
    ```

### 3. 実行
**管理者権限で開いたターミナル**で、以下のコマンドを実行します。
```bash
go run .
```

### 4. サーバーへの招待
[Discord Developer Portal](https://discord.com/developers/applications)の`OAuth2 > URL Generator`から、以下のスコープと権限を選択して招待リンクを生成してください。

* **SCOPES**: `bot`, `applications.commands`
* **BOT PERMISSIONS**:
    * `Send Messages`, `Embed Links`, `Read Messages/View Channels`
    * `Manage Channels` (チケット機能)
    * `Manage Roles` (チケット機能)
    * `Kick Members`, `Ban Members`, `Moderate Members`
    * `View Audit Log` (ログ機能)

---

## 📋 コマンドリスト

| コマンド | 説明 | 必要な権限 |
|:---|:---|:---|
| `/ping` | ボットの応答速度を測定します。 | 全員 |
| `/avatar` | ユーザーのアバターを表示します。| 全員 |
| `/embed` | カスタムEmbedメッセージを作成します。 | 全員 |
| `/ask` | AIに質問します。 | 全員 |
| `/calc` | 数式を計算します。 | 全員 |
| `/convert-power` | 工業MODの電力単位を相互変換します。 | 全員 |
| `/calc-stats` | ポケモンの実数値を詳細に計算します。 | 全員 |
| `/ticket-setup` | サポートチケット作成パネルを設置します。 | チャンネルの管理 |
| `/log-setup` | ログを送信するチャンネルを設定します。 | サーバーの管理 |
| `/kick` | ユーザーをサーバーから追放します。 | メンバーをキック |
| `/ban` | ユーザーをサーバーからBANします。 | メンバーをBAN |
| `/timeout` | ユーザーをタイムアウトさせます。 | メンバーをタイムアウト |

---
## 📜 ログ機能で記録されるイベント一覧
`/log-setup`で設定したチャンネルには、以下のイベントが発生した際にログが記録されます。
- メンバーの参加 / 退出
- メンバーのKick / BAN / Timeout (解除も含む)
- チャンネルの作成 / 削除
- メッセージの削除
- Webhookの更新

---

## ライセンス
このプロジェクトは [LGPL-3.0](LICENSE) の下で公開されています。