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

// サーバーごとの接続情報とプレイヤーを保存
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

        const connection = joinVoiceChannel({
            channelId: member.voice.channel.id,
            guildId: guild.id,
            adapterCreator: guild.voiceAdapterCreator,
        });
        
        // 接続状態の監視
        connection.on(VoiceConnectionStatus.Ready, () => {
            console.log('The connection has entered the Ready state - ready to play!');
        });
        
        connection.on(VoiceConnectionStatus.Disconnected, () => {
            console.log('Voice connection was disconnected.');
            // 必要に応じて再接続処理など
        });

        // play-dlでYouTubeのストリーム情報を取得
        let streamInfo = await play.stream(query, {
            quality: 1 // 0: low, 1: medium, 2: high
        });

        // オーディオプレイヤーを作成
        const audioPlayer = createAudioPlayer();
        const resource = createAudioResource(streamInfo.stream, {
            inputType: streamInfo.type
        });

        // プレイヤーにリソースをセットして再生
        audioPlayer.play(resource);
        connection.subscribe(audioPlayer);

        // 再生が開始されたら通知
        audioPlayer.on(AudioPlayerStatus.Playing, () => {
            textChannel.send(`🎵 再生中: **${streamInfo.video_details.title}**`);
        });

        // 再生が終了したら接続を切る
        audioPlayer.on(AudioPlayerStatus.Idle, () => {
             if (connection.state.status !== VoiceConnectionStatus.Destroyed) {
                connection.destroy();
                connections.delete(guildId);
            }
        });

        // サーバー情報を保存
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
        serverConnection.connection.destroy();
        connections.delete(guildId);
        return res.status(200).send({ message: '⏹️ 再生を停止しました。' });
    } else {
        return res.status(400).send({ error: '現在再生中ではありません。' });
    }
});

// スキップ機能はキュー管理が必要なため、このシンプルな実装では省略しています。

client.login(process.env.DISCORD_BOT_TOKEN);
app.listen(port, () => {
    console.log(`Music player server listening at http://localhost:${port}`);
});