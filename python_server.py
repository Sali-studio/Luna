import os
import flask
from flask import request, jsonify
import vertexai
from vertexai.preview.vision_models import ImageGenerationModel, VideoGenerationModel
from vertexai.generative_models import GenerativeModel

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
vertexai.init() # .envの認証情報を自動で読み込みま���

# --- モデルのロード ---
# 画像生成モデル
image_model = ImageGenerationModel.from_pretrained("imagen-4.0-generate-preview-06-06")
# 動画生成モデル (注意: モデル名はプレースホルダーです。正式なVEO-3のモデル名に置き換えてください)
video_model = VideoGenerationModel.from_pretrained("veo-1.0-generate-preview-05-20")

text_model = GenerativeModel("gemini-2.5-flash-preview-05-20")

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
        # 注意: generate_videosメソッドの引数はモデルによって異なる可能性があります
        video_data = video_model.generate(prompt=prompt)
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