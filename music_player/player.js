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

// サーバー起動時に一度だけ実行
player.on('ready', async () => {
    // YouTubeやSoundCloudなど、インストール済みの音源を全て読み込む
    await player.extractors.loadDefault();
});

// 曲の再生が始まったらメッセージを送信
player.on('trackStart', (queue, track) => {
    if (queue.metadata && queue.metadata.channel) {
        queue.metadata.channel.send(`🎵 再生中: **${track.title}**`);
    }
});

// エラーが発生した場合
player.on('error', (queue, error) => {
    console.log(`[Player Error] ${error.message}`);
    console.log(error);
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
        const searchResult = await player.search(query, { requestedBy: member.user });
        if (!searchResult.hasTracks()) {
            return res.status(404).send(`❌ トラックが見つかりませんでした: ${query}`);
        }
        
        // player.play() メソッドを使って再生を開始
        await player.play(voiceChannel, searchResult, {
            nodeOptions: {
                metadata: { channel: textChannel },
                volume: 80,
                leaveOnEmpty: true,
                leaveOnEmptyCooldown: 300000, // 5分
                leaveOnEnd: false,
            }
        });

        res.status(200).send(`✅ キューに追加しました。`);

    } catch (e) {
        console.error(e);
        return res.status(500).send(`エラーが発生しました: ${e.message}`);
    }
});

app.post('/skip', (req, res) => {
    const queue = player.getQueue(req.body.guildId);
    if (!queue || !queue.playing) return res.status(400).send('再生中の曲がありません。');
    const success = queue.skip();
    res.status(200).send(success ? '⏭️ スキップしました。' : '❌ スキップに失敗しました。');
});

app.post('/stop', (req, res) => {
    const queue = player.getQueue(req.body.guildId);
    if (!queue) return res.status(400).send('キューがありません。');
    queue.destroy();
    res.status(200).send('⏹️ 再生を停止し、キューをクリアしました。');
});

client.login(process.env.DISCORD_BOT_TOKEN).then(() => {
    console.log("Music Player Bot is online!");
    app.listen(port, () => {
        console.log(`Music player server listening at http://localhost:${port}`);
    });
});