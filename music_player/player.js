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

// プレーヤーがトラックの再生を開始したときのイベント
player.events.on('playerStart', (queue, track) => {
    queue.metadata.channel.send(`🎵 再生中: **${track.title}**`);
});

player.events.on('error', (queue, error) => {
    console.log(`[${queue.guild.name}] Error from queue: ${error.message}`);
});
player.events.on('connectionError', (queue, error) => {
    console.log(`[${queue.guild.name}] Error from connection: ${error.message}`);
});


client.on('ready', () => {
    console.log('Music Player Bot is online!');
});

// --- Goからのリクエストを待つWebサーバー ---
const app = express();
app.use(express.json());
const port = 8080; // Goと通信するためのポート

// `/play` エンドポイント
app.post('/play', async (req, res) => {
    const { guildId, channelId, query, userId } = req.body;

    if (!guildId || !channelId || !query || !userId) {
        return res.status(400).send('Missing required fields.');
    }

    const guild = client.guilds.cache.get(guildId);
    // 元のchannelIdはテキストチャンネルIDなので、メンバーのボイスチャンネルを取得する
    const member = await guild.members.fetch(userId);

    if (!member.voice.channel) {
        return res.status(400).send('User is not in a voice channel.');
    }
    
    // キューの取得または作成方法を変更
    const queue = player.nodes.create(guild, {
        metadata: {
            channel: guild.channels.cache.get(channelId) // テキストチャンネルをメタデータに保存
        },
        // 高音質化のためのオプション
        ytdlOptions: {
            quality: 'highestaudio',
            highWaterMark: 1 << 25
        },
        leaveOnEnd: false,
        leaveOnStop: true,
        leaveOnEmpty: true,
        leaveOnEmptyCooldown: 300000, // 5分
    });

    try {
        // ボイスチャンネルに接続
        if (!queue.connection) await queue.connect(member.voice.channel);

        const searchResult = await player.search(query, {
            requestedBy: member.user
        });

        if (!searchResult || !searchResult.tracks.length) return res.status(404).send('Track not found.');

        // キューにトラックを追加して再生
        searchResult.playlist ? queue.addTrack(searchResult.tracks) : queue.addTrack(searchResult.tracks[0]);
        if (!queue.isPlaying()) await queue.node.play();

        return res.status(200).send(`✅ キューに追加しました: **${searchResult.tracks[0].title}**`);
    } catch (e) {
        console.error(e);
        return res.status(500).send('Something went wrong.');
    }
});

// `/skip` エンドポイント
app.post('/skip', (req, res) => {
    const { guildId } = req.body;
    // ★★★ 修正箇所 ★★★
    const queue = player.nodes.get(guildId);
    if (!queue || !queue.isPlaying()) return res.status(400).send('No music is being played.');
    const success = queue.node.skip();
    return res.status(200).send(success ? '⏭️ スキップしました' : 'Something went wrong.');
});

// `/stop` エンドポイント
app.post('/stop', (req, res) => {
    const { guildId } = req.body;
    const queue = player.nodes.get(guildId);
    if (!queue || !queue.isPlaying()) return res.status(400).send('No music is being played.');
    // destroyではなくdeleteを使用
    queue.delete();
    return res.status(200).send('⏹️ 再生を停止しました');
});


// --- BotとWebサーバーの起動 ---
client.login(process.env.DISCORD_BOT_TOKEN); // GoのBotと同じトークンを使用
app.listen(port, () => {
    console.log(`Music player server listening at http://localhost:${port}`);
});