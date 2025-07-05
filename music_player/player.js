const express = require('express');
const { Client, GatewayIntentBits } = require('discord.js');
const { Player } = require('discord-player');
const { DefaultExtractors } = require('@discord-player/extractor'); 

// --- Discord Botのクライアントを初期化 ---
const client = new Client({
    intents: [
        GatewayIntentBits.Guilds,
        GatewayIntentBits.GuildVoiceStates
    ]
});

// --- 音楽プレーヤーを初期化 ---
const player = new Player(client);

// 非同期処理をまとめるためのメイン関数
async function main() {
    // 音源抽出器を読み込む
    await player.extractors.loadMulti(DefaultExtractors);

    // プレーヤーがトラックの再生を開始したときのイベント
    player.events.on('playerStart', (queue, track) => {
        queue.metadata.channel.send(`🎵 再生中: **${track.title}**`);
    });

    player.events.on('error', (queue, error) => {
        console.log(`[${queue.guild.name}] Error from queue: ${error.message}`);
        queue.metadata.channel.send('❌ 再生中にエラーが発生しました。');
    });
    player.events.on('connectionError', (queue, error) => {
        console.log(`[${queue.guild.name}] Error from connection: ${error.message}`);
        queue.metadata.channel.send('❌ ボイスチャンネルへの接続に失敗しました。');
    });

    client.on('ready', () => {
        console.log('Music Player Bot is online!');
    });

    // --- Goからのリクエストを待つWebサーバー ---
    const app = express();
    app.use(express.json());
    const port = 8080;

    // `/play` エンドポイント
    app.post('/play', async (req, res) => {
        const { guildId, channelId, query, userId } = req.body;

        if (!guildId || !channelId || !query || !userId) {
            return res.status(400).send('Missing required fields.');
        }

        const guild = client.guilds.cache.get(guildId);
        const member = await guild.members.fetch(userId);
        const textChannel = guild.channels.cache.get(channelId);

        if (!member.voice.channel) {
            return res.status(400).send('User is not in a voice channel.');
        }

        const queue = player.nodes.create(guild, {
            metadata: {
                channel: textChannel
            },
            ytdlOptions: {
                quality: 'highestaudio',
                highWaterMark: 1 << 25
            },
            leaveOnEnd: false,
            leaveOnStop: true,
            leaveOnEmpty: true,
            leaveOnEmptyCooldown: 300000,
        });

        try {
            if (!queue.connection) await queue.connect(member.voice.channel);

            const searchResult = await player.search(query, {
                requestedBy: member.user
            });

            if (!searchResult.hasTracks()) return res.status(404).send('❌ トラックが見つかりませんでした。');

            searchResult.playlist ? queue.addTrack(searchResult.tracks) : queue.addTrack(searchResult.tracks[0]);
            if (!queue.isPlaying()) await queue.node.play();

            return res.status(200).send(`✅ キューに追加しました: **${searchResult.tracks[0].title}**`);
        } catch (e) {
            console.error(e);
            return res.status(500).send('❌ 不明なエラーが発生しました。');
        }
    });

    // `/skip` エンドポイント
    app.post('/skip', (req, res) => {
        const { guildId } = req.body;
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
        queue.delete();
        return res.status(200).send('⏹️ 再生を停止しました');
    });


    // --- BotとWebサーバーの起動 ---
    client.login(process.env.DISCORD_BOT_TOKEN);
    app.listen(port, () => {
        console.log(`Music player server listening at http://localhost:${port}`);
    });
}

main();