import flask
from flask import request, jsonify
import yt_dlp

app = flask.Flask(__name__)

# yt-dlpで音声のURLを取得するための設定
YDL_OPTIONS = {
    'format': 'bestaudio/best',
    'noplaylist': 'True',
    'default_search': 'auto', # URLでない場合は検索する
}

@app.route('/get-stream-url', methods=['POST'])
def get_stream_url():
    """
    YouTubeのURLや検索ワードから、直接再生可能な音声ストリームのURLを取得する
    """
    data = request.get_json()
    query = data.get('query')
    if not query:
        return jsonify({'error': 'Query is required'}), 400

    try:
        # yt-dlpを使って情報を抽出
        with yt_dlp.YoutubeDL(YDL_OPTIONS) as ydl:
            info = ydl.extract_info(query, download=False)
            # 複数の動画情報が含まれている場合は最初のものを使用
            if 'entries' in info:
                info = info['entries'][0]
            
            # 音声ストリームのURLと曲のタイトルを返す
            return jsonify({
                'stream_url': info['url'],
                'title': info.get('title', '不明なタイトル')
            })
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500


if __name__ == '__main__':
    # Flaskサーバーをポート5002で起動
    app.run(host='0.0.0.0', port=5002)