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

// プレーヤーのイベントリスナー（エラーハンドリングなど）
player.on('error', (queue, error) => {
    console.log(`[${queue.guild.name}] Error from queue: ${error.message}`);
});
player.on('connectionError', (queue, error) => {
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
    const channel = guild.channels.cache.get(channelId);
    const member = await guild.members.fetch(userId);


    if (!channel || !member.voice.channel) {
        return res.status(400).send('User is not in a voice channel.');
    }

    try {
        const queue = player.createQueue(guild, {
             metadata: {
                channel: channel
            },
            // 高音質化のためのオプション
            ytdlOptions: {
                quality: 'highestaudio',
                highWaterMark: 1 << 25
            }
        });

        // ボイスチャンネルに接続
        if (!queue.connection) await queue.connect(member.voice.channel);

        const track = await player.search(query, {
            requestedBy: member.user
        }).then(x => x.tracks[0]);

        if (!track) return res.status(404).send('Track not found.');

        queue.play(track);

        return res.status(200).send(`🎵 Queued: **${track.title}**`);
    } catch (e) {
        console.error(e);
        return res.status(500).send('Something went wrong.');
    }
});

// `/skip` エンドポイント
app.post('/skip', (req, res) => {
    const { guildId } = req.body;
    const queue = player.getQueue(guildId);
    if (!queue || !queue.playing) return res.status(400).send('No music is being played.');
    const success = queue.skip();
    return res.status(200).send(success ? '⏭️ Skipped!' : 'Something went wrong.');
});

// `/stop` エンドポイント
app.post('/stop', (req, res) => {
    const { guildId } = req.body;
    const queue = player.getQueue(guildId);
    if (!queue || !queue.playing) return res.status(400).send('No music is being played.');
    queue.destroy();
    return res.status(200).send('⏹️ Stopped!');
});


// --- BotとWebサーバーの起動 ---
client.login(process.env.DISCORD_BOT_TOKEN); // GoのBotと同じトークンを使用
app.listen(port, () => {
    console.log(`Music player server listening at http://localhost:${port}`);
});