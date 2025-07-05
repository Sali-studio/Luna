const express = require('express');
const { Client, GatewayIntentBits } = require('discord.js');
const { Player } = require('discord-player');

// --- Discord Botのクライアントを初期化 ---
const client = new Client({
    intents: [
        GatewayIntentBits.Guilds,
        GatewayIntentBits.GuildVoiceStates
    ]
});

// --- 音楽プレーヤーを初期化 ---
const player = new Player(client);

// --- 詳細なデバッグログを出力するためのイベントリスナー ---

// デバッグメッセージ（ライブラリの内部動作）
player.on('debug', (queue, message) => {
    console.log(`[DEBUG] ${message}`);
});

// 曲の再生が開始されたとき
player.on('trackStart', (queue, track) => {
    console.log(`[INFO] ▶️ Playing: ${track.title} in ${queue.guild.name}`);
    queue.metadata.channel.send(`▶️ **${track.title}** の再生を開始しました。`);
});

// 曲がキューに追加されたとき
player.on('trackAdd', (queue, track) => {
    console.log(`[INFO] ➕ Track ${track.title} added to queue.`);
    queue.metadata.channel.send(`➕ **${track.title}** をキューに追加しました。`);
});

// キューが空になったとき
player.on('queueEnd', (queue) => {
    console.log(`[INFO] ✅ Queue finished in ${queue.guild.name}.`);
});

// ボイスチャンネルから切断されたとき
player.on('clientDisconnect', (queue) => {
    console.log(`[WARN] I was disconnected from the voice channel in ${queue.guild.name}.`);
});

// 一般的なエラー
player.on('error', (queue, error) => {
    console.error(`[ERROR] General player error: ${error.message}`);
    console.error(error);
});

// 接続に関するエラー
player.on('connectionError', (queue, error) => {
    console.error(`[ERROR] Connection error: ${error.message}`);
    console.error(error);
});


client.on('ready', () => {
    console.log('Music Player Bot is online!');
    // すべてのExtractorをロード
    player.extractors.loadDefault();
});

// --- Goからのリクエストを待つWebサーバー ---
const app = express();
app.use(express.json());
const port = 8080;

// `/play` エンドポイント
app.post('/play', async (req, res) => {
    console.log('[API] /play endpoint hit with body:', req.body);
    const { guildId, channelId, query, userId } = req.body;

    if (!guildId || !channelId || !query || !userId) {
        console.error('[API-ERROR] Missing required fields.');
        return res.status(400).send('Missing required fields.');
    }

    const guild = client.guilds.cache.get(guildId);
    if (!guild) {
         console.error(`[API-ERROR] Guild not found for ID: ${guildId}`);
         return res.status(404).send('Guild not found.');
    }
    const member = await guild.members.fetch(userId).catch(() => null);
     if (!member) {
        console.error(`[API-ERROR] Member not found for ID: ${userId}`);
        return res.status(404).send('Member not found.');
    }
    const voiceChannel = member.voice.channel;
    if (!voiceChannel) {
        console.error('[API-ERROR] User is not in a voice channel.');
        return res.status(400).send('User is not in a voice channel.');
    }
    
    // playメソッドを呼び出す直前にログを出力
    console.log(`[PLAYER] Attempting to play "${query}" in guild ${guild.name}`);

    try {
        const result = await player.play(voiceChannel, query, {
            nodeOptions: {
                metadata: {
                    channel: req.body.textChannel, // メッセージを送信するテキストチャンネル
                    requestedBy: member.user,
                },
                // 高音質化のためのオプション
                ytdlOptions: {
                    quality: 'highestaudio',
                    highWaterMark: 1 << 25,
                },
            }
        });
        
        console.log(`[PLAYER] player.play() completed. Sending success response.`);
        // 成功した場合は、Go側に再生が開始されたことを伝える
        return res.status(200).json({ message: `✅ Queued: **${result.track.title}**` });

    } catch (e) {
        console.error('[PLAYER-ERROR] Failed to play track:', e);
        return res.status(500).send('Could not play the track.');
    }
});

// その他のエンドポイント（skip, stopなど）も同様にログを追加できます
app.post('/skip', (req, res) => {
    const { guildId } = req.body;
    const queue = player.getQueue(guildId);
    if (!queue || !queue.playing) return res.status(400).send("No music is being played!");
    const success = queue.skip();
    res.status(200).send(success ? "⏭️ Skipped!" : "❌ Something went wrong!");
});

app.post('/stop', (req, res) => {
    const { guildId } = req.body;
    const queue = player.getQueue(guildId);
    if (!queue) return res.status(400).send("No music queue found.");
    queue.destroy();
    res.status(200).send("⏹️ Stopped the player!");
});

// --- サーバー起動 ---
client.login(process.env.DISCORD_BOT_TOKEN).then(() => {
    app.listen(port, () => {
        console.log(`Music player server listening at http://localhost:${port}`);
    });
});