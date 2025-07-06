const express = require('express');
const { Client, GatewayIntentBits } = require('discord.js');
const {
    joinVoiceChannel,
    createAudioPlayer,
    createAudioResource,
    AudioPlayerStatus,
    VoiceConnectionStatus
} = require('@discordjs/voice');
const play = require('play-dl');

const client = new Client({
    intents: [
        GatewayIntentBits.Guilds,
        GatewayIntentBits.GuildVoiceStates
    ]
});

// サーバーごとの接続情報を保存
const serverQueues = new Map();

client.on('ready', () => {
    console.log('Music Player Bot is online!');
});

const app = express();
app.use(express.json());
const port = 8080;

app.post('/play', async (req, res) => {
    const { guildId, channelId, query, userId } = req.body;
    if (!guildId || !channelId || !query || !userId) {
        return res.status(400).send({ error: 'リクエスト情報が不足しています。' });
    }

    try {
        const guild = await client.guilds.fetch(guildId);
        const member = await guild.members.fetch(userId);
        const textChannel = await guild.channels.fetch(channelId);

        if (!member.voice.channel) {
            return res.status(400).send({ error: 'まずボイスチャンネルに参加してください。' });
        }
        
        const queue = serverQueues.get(guildId);
        
        // 1. play-dlで動画を検索
        const searchResult = await play.search(query, { limit: 1 });
        if (searchResult.length === 0) {
            return res.status(404).send({ error: 'トラックが見つかりませんでした。' });
        }
        const track = searchResult[0];

        if (!queue) {
            // キューがない場合は、新しいキューを作成して接続
            const newQueue = {
                voiceChannel: member.voice.channel,
                textChannel: textChannel,
                connection: null,
                player: createAudioPlayer(),
                songs: [track]
            };

            serverQueues.set(guildId, newQueue);

            try {
                newQueue.connection = joinVoiceChannel({
                    channelId: member.voice.channel.id,
                    guildId: guild.id,
                    adapterCreator: guild.voiceAdapterCreator,
                });

                newQueue.connection.subscribe(newQueue.player);
                playNext(guildId);
                res.status(200).send(`✅ **${track.title}** の再生を開始します。`);

            } catch (err) {
                console.error(err);
                serverQueues.delete(guildId);
                return res.status(500).send({ error: 'ボイスチャンネルへの接続に失敗しました。' });
            }
        } else {
            // 既にキューがある場合は、曲を追加
            queue.songs.push(track);
            return res.status(200).send(`✅ **${track.title}** をキューに追加しました。`);
        }

    } catch (e) {
        console.error('Error in /play route:', e);
        return res.status(500).send({ error: `エラーが発生しました: ${e.message}` });
    }
});

async function playNext(guildId) {
    const queue = serverQueues.get(guildId);
    if (!queue) return;
    if (queue.songs.length === 0) {
        // キューが空になったらVCから切断
        if (queue.connection) {
            queue.connection.destroy();
        }
        serverQueues.delete(guildId);
        return;
    }

    const track = queue.songs.shift();

    try {
        const stream = await play.stream(track.url);
        const resource = createAudioResource(stream.stream, { inputType: stream.type });
        
        queue.player.play(resource);
        queue.textChannel.send(`🎵 再生中: **${track.title}**`);

        queue.player.once(AudioPlayerStatus.Idle, () => {
            playNext(guildId);
        });
    } catch (error) {
        console.error(`Error playing track: ${error}`);
        queue.textChannel.send(`❌ **${track.title}** の再生中にエラーが発生しました。`);
        playNext(guildId); // 次の曲へ
    }
}

app.post('/stop', (req, res) => {
    const guildId = req.body.guildId;
    const queue = serverQueues.get(guildId);

    if (queue) {
        queue.songs = []; // キューを空にする
        if (queue.player) queue.player.stop();
        if (queue.connection) queue.connection.destroy();
        serverQueues.delete(guildId);
        res.status(200).send({ message: '⏹️ 再生を停止しました。' });
    } else {
        res.status(400).send({ error: '現在再生中ではありません。' });
    }
});

app.post('/skip', (req, res) => {
    const guildId = req.body.guildId;
    const queue = serverQueues.get(guildId);
    if (!queue || queue.songs.length === 0) {
        return res.status(400).send({ error: 'スキップする曲がありません。' });
    }
    // プレイヤーを停止させると、'Idle'イベントが発火して次の曲が再生される
    queue.player.stop(); 
    res.status(200).send({ message: '⏭️ スキップしました。' });
});

client.login(process.env.DISCORD_BOT_TOKEN);
app.listen(port, () => {
    console.log(`Music player server listening at http://localhost:${port}`);
});