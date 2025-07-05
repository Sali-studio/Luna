# ai_server.py

import os
import flask
from flask import request, jsonify
import vertexai
from vertexai.preview.vision_models import ImageGenerationModel
from vertexai.generative_models import GenerativeModel

import time

# 一時的に画像を保存するフォルダ
IMG_DIR = "generated_images"
if not os.path.exists(IMG_DIR):
    os.makedirs(IMG_DIR)

# --- FlaskアプリとVertex AIの初期化 ---
app = flask.Flask(__name__)
vertexai.init() # .envの認証情報を自動で読み込みます

# --- モデルのロード ---
# 画像生成モデル
image_model = ImageGenerationModel.from_pretrained("imagegeneration@005") 

# テキスト生成モデル (最新のGemini 2.5 Flash)
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
        
        server_url = f"http://localhost:5001/images/{filename}"
        print(f"✅ Image saved: {filepath}")
        
        return jsonify({'image_url': server_url})

    except Exception as e:
        print(f"❌ Error generating image: {e}")
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

# --- サーバーの起動 ---
if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)