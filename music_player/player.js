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
async function setupPlayer() {
    await player.extractors.loadDefault();
    console.log("Audio extractors loaded successfully!");
}

setupPlayer();

// 再生開始時のイベント
player.on('trackStart', (queue, track) => {
    if (queue.metadata.channel) {
        queue.metadata.channel.send(`🎵 再生中: **${track.title}**`);
    }
});

// エラー発生時のイベント
player.on('error', (queue, error) => {
    console.error(`[Player Error] ${error.message}`, error);
    if (queue.metadata.channel) {
        queue.metadata.channel.send(`❌ エラーが発生しました: ${error.message}`);
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
    if (!member || !member.voice.channel) {
        return res.status(400).send('まずボイスチャンネルに参加してください。');
    }

    const textChannel = guild.channels.cache.get(channelId);
    if (!textChannel) return res.status(404).send('テキストチャンネルが見つかりません。');
    
    try {
        const queue = player.nodes.create(guild, {
            metadata: { channel: textChannel },
            leaveOnEmpty: true,
            leaveOnEmptyCooldown: 300000,
            leaveOnEnd: false,
            volume: 80
        });

        if (!queue.connection) await queue.connect(member.voice.channel);

        const searchResult = await player.search(query, { requestedBy: member.user });
        if (!searchResult.hasTracks()) {
            return res.status(404).send(`❌ トラックが見つかりませんでした: ${query}`);
        }

        queue.addTrack(searchResult.tracks[0]);
        if (!queue.isPlaying()) await queue.node.play();

        return res.status(200).send(`✅ **${searchResult.tracks[0].title}** をキューに追加しました。`);
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