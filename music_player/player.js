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

// このasync関数内で、playerの準備が完了してからサーバーを起動します。
async function start() {
    // YouTubeやSoundCloudなど、インストール済みの音源を全て読み込みます。
    await player.extractors.loadDefault();

    // 曲の再生が始まったらメッセージを送信
    player.on('trackStart', (queue, track) => {
        if (queue.metadata && queue.metadata.channel) {
            queue.metadata.channel.send(`🎵 再生中: **${track.title}**`);
        }
    });

    // エラーが発生した場合
    player.on('error', (queue, error) => {
        console.error(`[Player Error] ${error.message}`, error);
        if (queue.metadata && queue.metadata.channel) {
            queue.metadata.channel.send(`❌ 再生中にエラーが発生しました: ${error.message}`);
        }
    });

    const app = express();
    app.use(express.json());
    const port = 8080;

    app.post('/play', async (req, res) => {
        const { guildId, channelId, query, userId } = req.body;

        if (!guildId || !channelId || !query || !userId) {
            return res.status(400).send('リクエストに必要な情報が不足しています。');
        }

        const guild = client.guilds.cache.get(guildId);
        if (!guild) return res.status(404).send('Botがそのサーバーに参加していません。');
        
        const member = await guild.members.fetch(userId).catch(() => null);
        if (!member) return res.status(404).send('ユーザーが見つかりません。');

        const voiceChannel = member.voice.channel;
        if (!voiceChannel) {
            return res.status(400).send('まずボイスチャンネルに参加してください。');
        }

        const textChannel = guild.channels.cache.get(channelId);
        if (!textChannel) return res.status(404).send('テキストチャンネルが見つかりません。');
        
        try {
            // player.play() を使うと、キューの作成や接続を自動で行います。
            const { track } = await player.play(voiceChannel, query, {
                nodeOptions: {
                    metadata: { channel: textChannel },
                    volume: 80,
                    leaveOnEmpty: true,
                    leaveOnEmptyCooldown: 300000,
                    leaveOnEnd: false,
                }
            });
            
            return res.status(200).send(`✅ **${track.title}** をキューに追加しました。`);

        } catch (e) {
            console.error(e);
            return res.status(500).send(`エラーが発生しました: ${e.message}`);
        }
    });

    app.post('/skip', (req, res) => {
        const queue = player.nodes.get(req.body.guildId);
        if (!queue || !queue.isPlaying()) return res.status(400).send('再生中の曲がありません。');
        queue.node.skip();
        res.status(200).send('⏭️ スキップしました。');
    });

    app.post('/stop', (req, res) => {
        const queue = player.nodes.get(req.body.guildId);
        if (!queue) return res.status(400).send('キューがありません。');
        queue.delete();
        res.status(200).send('⏹️ 再生を停止し、キューをクリアしました。');
    });

    client.login(process.env.DISCORD_BOT_TOKEN).then(() => {
        console.log("Music Player Bot is online!");
        app.listen(port, () => {
            console.log(`Music player server listening at http://localhost:${port}`);
        });
    });
}

// サーバーを起動
start();