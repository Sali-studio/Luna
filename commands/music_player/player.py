import flask
from flask import request, jsonify
import yt_dlp
import os

app = flask.Flask(__name__)

@app.route('/get-stream-url', methods=['POST'])
def get_stream_url():
    data = request.get_json()
    query = data.get('query')
    if not query:
        return jsonify({'error': 'Query is required'}), 400

    try:
        ydl_opts = {
            'format': 'bestaudio/best',
            'noplaylist': True,
            'default_search': 'auto',
        }

        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            info = ydl.extract_info(query, download=False)
            if 'entries' in info:
                info = info['entries'][0]
            
            return jsonify({
                'stream_url': info['url'],
                'title': info.get('title', '不明なタイトル')
            })
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5002)