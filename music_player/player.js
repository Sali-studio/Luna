console.log("--- music player debug mode ---");

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
    // Extractorのロード処理をtry...catchで囲み、エラーを捕捉する
    try {
        console.log("Extractorのロードを開始します...");
        await player.extractors.loadDefault();
        console.log("✅ Extractorのロードが正常に完了しました！");
    } catch (error) {
        console.error("❌ Extractorのロード中に致命的なエラーが発生しました:", error);
        // エラーが発生したらプロセスを終了し、問題を明確にする
        process.exit(1); 
    }

    player.on('trackStart', (queue, track) => {
        if (queue.metadata && queue.metadata.channel) {
            queue.metadata.channel.send(`🎵 再生中: **${track.title}**`);
        }
    });

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
            return res.status(400).send('リクエスト情報が不足しています。');
        }

        const guild = client.guilds.cache.get(guildId);
        if (!guild) return res.status(404).send('サーバーが見つかりません。');
        
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
}

start();