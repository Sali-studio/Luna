import os
import flask
from flask import request, jsonify
import vertexai
from vertexai.preview.vision_models import ImageGenerationModel
from vertexai.generative_models import GenerativeModel, GenerationResponse

import time

# 一時的にファイルを保存するフォルダ
IMG_DIR = "generated_images"
VIDEO_DIR = "generated_videos"
if not os.path.exists(IMG_DIR):
    os.makedirs(IMG_DIR)
if not os.path.exists(VIDEO_DIR):
    os.makedirs(VIDEO_DIR)

# --- FlaskアプリとVertex AIの初期化 ---
app = flask.Flask(__name__)
vertexai.init() # .envの認証情報を自動で読み込みます

# --- モデルのロード ---
# 画像生成モデル
image_model = ImageGenerationModel.from_pretrained("imagen-4.0-generate-preview-06-06") 

# テキスト生成モデル
text_model = GenerativeModel("gemini-2.5-flash-preview-05-20")

# 動画生成モデル
# VEOはまだプレビュー段階のため、GenerativeModel経由で呼び出します。
# 注意: モ���ル名はプレースホルダーです。正式なVEO-3のモデル名に置き換えてください。
video_model = GenerativeModel("veo-1.0-generate-preview-05-20")

# --- APIエンドポイントの定義 ---
# 画像生成用のエンドポイント
@app.route('/generate-image', methods=['POST'])
def generate_image():
    data = request.get_json()
    if not data or 'prompt' not in data:
        return jsonify({'error': 'prompt is required'}), 400

    prompt = data['prompt']
    print(f"✅ Received Image prompt: {prompt}")

    try:
        print("⏳ Generating image...")
        images = image_model.generate_images(prompt=prompt, number_of_images=1)
        image_data = images[0]._image_bytes
        print("✅ Image generated.")

        filename = f"{int(time.time())}.png"
        filepath = os.path.join(IMG_DIR, filename)
        
        with open(filepath, "wb") as f:
            f.write(image_data)

        print(f"✅ Image saved: {filepath}")
        
        return jsonify({'image_path': os.path.abspath(filepath)})

    except Exception as e:
        print(f"❌ Error generating image: {e}")
        return jsonify({'error': str(e)}), 500

# 動画生成用のエンドポイント
@app.route('/generate-video', methods=['POST'])
def generate_video():
    data = request.get_json()
    if not data or 'prompt' not in data:
        return jsonify({'error': 'prompt is required'}), 400

    prompt = data['prompt']
    print(f"✅ Received Video prompt: {prompt}")

    try:
        print("⏳ Generating video...")
        response: GenerationResponse = video_model.generate_content([prompt])
        video_data = response.candidates[0].content.parts[0].blob.data

        print("✅ Video generated.")

        filename = f"{int(time.time())}.mp4"
        filepath = os.path.join(VIDEO_DIR, filename)
        
        with open(filepath, "wb") as f:
            f.write(video_data)

        print(f"✅ Video saved: {filepath}")
        
        return jsonify({'video_path': os.path.abspath(filepath)})

    except Exception as e:
        print(f"❌ Error generating video: {e}")
        return jsonify({'error': str(e)}), 500

# テキスト生成用のエンドポイント
@app.route('/generate-text', methods=['POST'])
def generate_text():
    data = request.get_json()
    if not data or 'prompt' not in data:
        return jsonify({'error': 'prompt is required'}), 400

    prompt = data['prompt']
    print(f"✅ Received Text prompt: {prompt}")

    try:
        print("⏳ Generating text...")
        response = text_model.generate_content(prompt)
        print("✅ Text generated.")
        return jsonify({'text': response.text})

    except Exception as e:
        print(f"❌ Error generating text: {e}")
        return jsonify({'error': str(e)}), 500


# 画像を配信するためのエンドポイント
@app.route('/images/<filename>')
def get_image(filename):
    return flask.send_from_directory(IMG_DIR, filename)

# 動画を配信するためのエンドポイント
@app.route('/videos/<filename>')
def get_video(filename):
    return flask.send_from_directory(VIDEO_DIR, filename)

# --- サーバーの起動 ---
if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)
