const express = require('express');
const { Client, GatewayIntentBits } = require('discord.js');
const { Player } = require('discord-player');

const client = new Client({
    intents: [
        GatewayIntentBits.Guilds,
        GatewayIntentBits.GuildVoiceStates
    ]
});

const player = new Player(client);

// BOTがDiscordに接続完了した後に、音楽機能の準備を開始
client.on('ready', async () => {
    try {
        await player.extractors.loadDefault();
        console.log('Music Player Bot is online and extractors are loaded!');
    } catch (error) {
        console.error('Failed to load extractors:', error);
    }
});

// 再生開始時のイベント
player.on('trackStart', (queue, track) => {
    if (queue.metadata && queue.metadata.channel) {
        queue.metadata.channel.send(`🎵 再生中: **${track.title}**`);
    }
});

// エラー発生時のイベント
player.on('error', (queue, error) => {
    console.error(`[Player Error] ${error.message}`);
    if (queue.metadata && queue.metadata.channel) {
        queue.metadata.channel.send(`❌ エラーが発生しました: ${error.message}`);
    }
});

const app = express();
app.use(express.json());
const port = 8080;

app.post('/play', async (req, res) => {
    const { guildId, channelId, query, userId } = req.body;

    if (!guildId || !channelId || !query || !userId) {
        return res.status(400).send({ error: 'リクエストに必要な情報が不足しています。' });
    }

    try {
        const guild = await client.guilds.fetch(guildId);
        const member = await guild.members.fetch(userId);
        const textChannel = await guild.channels.fetch(channelId);

        if (!member.voice.channel) {
            return res.status(400).send({ error: 'まずボイスチャンネルに参加してください。' });
        }
        
        const searchResult = await player.search(query, { requestedBy: member.user });
        if (!searchResult.hasTracks()) {
            return res.status(404).send({ error: `❌ トラックが見つかりませんでした: ${query}` });
        }

        // キューの取得または作成
        const queue = player.nodes.create(guild, {
            metadata: { channel: textChannel },
            leaveOnEmpty: true,
            leaveOnEmptyCooldown: 300000,
            leaveOnEnd: false,
            volume: 80
        });

        // 接続していなければ接続
        if (!queue.connection) await queue.connect(member.voice.channel);

        // トラックを追加して再生
        queue.addTrack(searchResult.tracks[0]);
        if (!queue.isPlaying()) await queue.node.play();

        return res.status(200).send({ message: `✅ **${searchResult.tracks[0].title}** をキューに追加しました。` });
    } catch (e) {
        console.error('Error in /play route:', e);
        return res.status(500).send({ error: `エラーが発生しました: ${e.message}` });
    }
});

app.post('/skip', (req, res) => {
    const queue = player.nodes.get(req.body.guildId);
    if (!queue || !queue.isPlaying()) return res.status(400).send({ error: '再生中の曲がありません。' });
    queue.node.skip();
    res.status(200).send({ message: '⏭️ スキップしました。' });
});

app.post('/stop', (req, res) => {
    const queue = player.nodes.get(req.body.guildId);
    if (!queue) return res.status(400).send({ error: 'キューがありません。' });
    queue.delete();
    res.status(200).send({ message: '⏹️ 再生を停止し、キューをクリアしました。' });
});

// BOTをログインさせ、その後Webサーバーを起動
client.login(process.env.DISCORD_BOT_TOKEN);

app.listen(port, () => {
    console.log(`Music player server listening at http://localhost:${port}`);
});