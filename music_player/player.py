import flask
from flask import request, jsonify
import discord
from discord.ext import commands
import yt_dlp
import asyncio
import threading

# --- 初期設定 ---
intents = discord.Intents.default()
intents.message_content = True
bot = commands.Bot(command_prefix="!", intents=intents)
app = flask.Flask(__name__)

# yt-dlpの設定
YDL_OPTIONS = {'format': 'bestaudio/best', 'noplaylist': 'True', 'outtmpl': '%(extractor)s-%(id)s-%(title)s.%(ext)s'}
FFMPEG_OPTIONS = {'before_options': '-reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 5', 'options': '-vn'}

# サーバーごとの音楽キュー
queues = {}

# --- Botの非同期処理 ---
@bot.event
async def on_ready():
    print(f'Music Player Logged in as {bot.user}')

async def play_next(guild_id):
    """次の曲を再生する"""
    if guild_id in queues and queues[guild_id]:
        # 次の曲の情報を取得
        url = queues[guild_id].pop(0)
        
        # 音声クライアントを取得
        voice_client = discord.utils.get(bot.voice_clients, guild_id=guild_id)
        if not voice_client:
            return

        # 音源を取得
        with yt_dlp.YoutubeDL(YDL_OPTIONS) as ydl:
            info = ydl.extract_info(url, download=False)
            audio_url = info['url']

        # 再生
        source = await discord.FFmpegOpusAudio.from_probe(audio_url, **FFMPEG_OPTIONS)
        voice_client.play(source, after=lambda e: asyncio.run_coroutine_threadsafe(play_next(guild_id), bot.loop).result())

# --- FlaskのAPIエンドポイント ---
@app.route('/play', methods=['POST'])
def play_song():
    data = request.get_json()
    guild_id = int(data['guild_id'])
    channel_id = int(data['channel_id'])
    query = data['query']

    async def _play():
        # ボイスチャンネルに接続
        guild = bot.get_guild(guild_id)
        if not guild:
            return jsonify({'error': 'Guild not found'}), 404
        
        voice_client = discord.utils.get(bot.voice_clients, guild=guild)
        if not voice_client:
            channel = guild.get_channel(channel_id)
            if not channel:
                return jsonify({'error': 'Channel not found'}), 404
            voice_client = await channel.connect()

        # キューに追加
        if guild_id not in queues:
            queues[guild_id] = []
        queues[guild_id].append(query)

        # 再生中でなければ再生開始
        if not voice_client.is_playing():
            await play_next(guild_id)

    future = asyncio.run_coroutine_threadsafe(_play(), bot.loop)
    future.result()
    return jsonify({'status': 'added to queue'})


@app.route('/skip', methods=['POST'])
def skip_song():
    data = request.get_json()
    guild_id = int(data['guild_id'])
    
    voice_client = discord.utils.get(bot.voice_clients, guild_id=guild_id)
    if voice_client and voice_client.is_playing():
        voice_client.stop()
        return jsonify({'status': 'skipped'})
    return jsonify({'error': 'Not playing anything'}), 400


@app.route('/stop', methods=['POST'])
def stop_music():
    data = request.get_json()
    guild_id = int(data['guild_id'])

    async def _stop():
        voice_client = discord.utils.get(bot.voice_clients, guild_id=guild_id)
        if voice_client:
            # キューをクリアして切断
            if guild_id in queues:
                queues[guild_id] = []
            await voice_client.disconnect()

    future = asyncio.run_coroutine_threadsafe(_stop(), bot.loop)
    future.result()
    return jsonify({'status': 'stopped and disconnected'})

# --- サーバーの起動 ---
def run_flask():
    app.run(host='0.0.0.0', port=5002)

def run_bot():
    # .envからTOKENを読み込む
    from dotenv import load_dotenv
    import os
    load_dotenv()
    bot_token = os.getenv("DISCORD_BOT_TOKEN")
    if bot_token:
        bot.run(bot_token)
    else:
        print("Error: DISCORD_BOT_TOKEN not found in .env file")

if __name__ == '__main__':
    # Flaskサーバーを別スレッドで起動
    flask_thread = threading.Thread(target=run_flask)
    flask_thread.daemon = True
    flask_thread.start()

    # Discord Botをメインスレッドで起動
    run_bot()