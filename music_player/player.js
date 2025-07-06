const express = require('express');
const { Client, GatewayIntentBits } = require('discord.js');
const { joinVoiceChannel, createAudioPlayer, createAudioResource, AudioPlayerStatus, VoiceConnectionStatus } = require('@discordjs/voice');
const play = require('play-dl');

const client = new Client({
    intents: [
        GatewayIntentBits.Guilds,
        GatewayIntentBits.GuildVoiceStates
    ]
});

// サーバーごとの接続情報を保存
const connections = new Map();

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

        // 1. play-dlで動画を検索 (URLでも検索ワードでもOK)
        const searchResults = await play.search(query, {
            limit: 1
        });

        if (searchResults.length === 0) {
            return res.status(404).send({ error: 'トラックが見つかりませんでした。' });
        }
        
        // 検索結果の最初の動画を使用
        const video = searchResults[0];

        // 2. 検索結果のURLからストリーム情報を取得
        const stream = await play.stream(video.url);

        const connection = joinVoiceChannel({
            channelId: member.voice.channel.id,
            guildId: guild.id,
            adapterCreator: guild.voiceAdapterCreator,
        });

        const audioPlayer = createAudioPlayer();
        const resource = createAudioResource(stream.stream, {
            inputType: stream.type
        });

        audioPlayer.play(resource);
        connection.subscribe(audioPlayer);

        // 再生が開始されたら通知
        audioPlayer.on(AudioPlayerStatus.Playing, () => {
            textChannel.send(`🎵 再生中: **${video.title}**`);
        });
        
        audioPlayer.on(AudioPlayerStatus.Idle, () => {
             if (connection.state.status !== VoiceConnectionStatus.Destroyed) {
                connection.destroy();
                connections.delete(guildId);
            }
        });

        connections.set(guildId, { connection, audioPlayer });

        return res.status(200).send({ message: '再生リクエストを受け付けました。' });

    } catch (e) {
        console.error('Error in /play route:', e);
        return res.status(500).send({ error: `エラーが発生しました: ${e.message}` });
    }
});

app.post('/stop', (req, res) => {
    const guildId = req.body.guildId;
    const serverConnection = connections.get(guildId);

    if (serverConnection && serverConnection.connection) {
        if(serverConnection.audioPlayer) serverConnection.audioPlayer.stop();
        if(serverConnection.connection) serverConnection.connection.destroy();
        connections.delete(guildId);
        return res.status(200).send({ message: '⏹️ 再生を停止しました。' });
    } else {
        return res.status(400).send({ error: '現在再生中ではありません。' });
    }
});


client.login(process.env.DISCORD_BOT_TOKEN);
app.listen(port, () => {
    console.log(`Music player server listening at http://localhost:${port}`);
});