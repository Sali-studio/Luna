# 🌙 Luna 1.5.3

Luna is a multi-functional Discord bot built with Go, designed to streamline server management and enhance community engagement.

[![Go Version](https://img.shields.io/badge/Go-1.18%2B-blue.svg)](https://golang.org/)
[![DiscordGo](https://img.shields.io/badge/lib-DiscordGo-blue.svg)](https://github.com/bwmarrin/discordgo)
[![License](https://img.shields.io/badge/License-LGPL--3.0-blue.svg)](LICENSE)

---

## ✨ Features

Luna is packed with a variety of features to improve your Discord server.

* **🤖 AI Chat**: Integrates with the Google Gemini API to answer user questions.
* **🎫 Ticket System**: A complete support ticket system using buttons and modals to create private channels for user inquiries.
* **🛡️ Moderation**: Simple and effective moderation commands, including Kick, Ban, and Timeout.
* **📈 Advanced Logging**: A superior audit log that records various server events in a specific channel with beautifully formatted embeds.
* **⚙️ Utilities**: Handy utility commands to provide useful information.
* **📝 Modular Design**: Features are organized by command, making maintenance and the addition of new features easy.

---

## 🚀 Installation

### 1. Prerequisites
* [Go](https://go.dev/dl/) (version 1.18 or higher) must be installed.

### 2. Configuration
1.  Clone or download this repository.
    ```bash
    git clone [https://github.com/pepeyukke/luna.git](https://github.com/pepeyukke/luna.git)
    cd luna
    ```
2.  Install the necessary dependencies.
    ```bash
    go mod tidy
    ```
3.  Set the required API keys as environment variables.

    **Windows (PowerShell)**
    ```powershell
    # Your Discord Bot Token
    $env:DISCORD_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"

    # Your Gemini API Key from Google AI Studio
    $env:GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
    ```
    **macOS / Linux**
    ```bash
    export DISCORD_BOT_TOKEN="YOUR_BOT_TOKEN_HERE"
    export GEMINI_API_KEY="YOUR_GEMINI_API_KEY_HERE"
    ```

### 3. Running the Bot
Run the following command from the project's root directory. It is recommended to run this in a terminal with **Administrator privileges**.
```bash
go run .
```

### 4. Inviting the Bot to Your Server
Generate an invite link from the `OAuth2 > URL Generator` page in your [Discord Developer Portal](https://discord.com/developers/applications). Select the following scopes and permissions:

* **SCOPES**: `bot`, `applications.commands`
* **BOT PERMISSIONS**:
    * `Send Messages`
    * `Embed Links`
    * `Read Messages/View Channels`
    * `Manage Channels` (Required for Ticket System)
    * `Manage Roles` (Required for Ticket System)
    * `Kick Members`
    * `Ban Members`
    * `Moderate Members` (Required for Timeout)
    * `View Audit Log` (Required for Logging System)

---

## 📋 Command List

| Command         | Description                                        | Permissions Required        |
|:----------------|:---------------------------------------------------|:----------------------------|
| `/ping`         | Measures the bot's response time (latency).        | Everyone                    |
| `/avatar`       | Displays your avatar or a specified user's avatar. | Everyone                    |
| `/embed`        | Creates a custom embed message with your content.  | Everyone                    |
| `/ask`          | Asks a question to the AI.                         | Everyone                    |
| `/ticket-setup` | Sets up the panel for creating support tickets.    | Manage Channels             |
| `/log-setup`    | Sets the channel for logging server events.        | Manage Server               |
| `/kick`         | Kicks a user from the server.                      | Kick Members                |
| `/ban`          | Bans a user from the server.                       | Ban Members                 |
| `/timeout`      | Times out a user for a specified duration.         | Moderate Members            |

---
## 📜 Logged Events
When configured with `/log-setup`, the following events will be logged:
- Member Join / Leave
- Member Kick / Ban / Timeout (including removal)
- Channel Create / Delete
- Message Delete
- Webhook Update

---

## License
This project is licensed under the [LGPL-3.0](LICENSE).