import os
import flask
from flask import request, jsonify
import vertexai
from vertexai.preview.vision_models import ImageGenerationModel
from vertexai.generative_models import GenerativeModel, Part

import time
import requests

# 一時的に画像を保存するフォルダ
IMG_DIR = "generated_images"
if not os.path.exists(IMG_DIR):
    os.makedirs(IMG_DIR)

# --- FlaskアプリとVertex AIの初期化 ---
app = flask.Flask(__name__)
vertexai.init() # .envの認証情報を自動で読み込みます

# --- モデルのロード ---
# 画像生成モデル
image_model = ImageGenerationModel.from_pretrained("imagen-4.0-generate-preview-06-06") 
# テキスト生成・画像認識モデル (多モーダル)
multimodal_model = GenerativeModel("gemini-2.5-pro")

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
        
        # 修正：URLの代わりに、保存したファイルの絶対パスを返す
        return jsonify({'image_path': os.path.abspath(filepath)})

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
        response = multimodal_model.generate_content(prompt)
        print("✅ Text generated.")
        return jsonify({'text': response.text})

    except Exception as e:
        print(f"❌ Error generating text: {e}")
        return jsonify({'error': str(e)}), 500

# ★★★ 新しいエンドポイント: 画像認識 ★★★
@app.route('/describe-image', methods=['POST'])
def describe_image():
    data = request.get_json()
    if not data or 'image_url' not in data:
        return jsonify({'error': 'image_url is required'}), 400

    image_url = data['image_url']
    print(f"✅ Received Image URL: {image_url}")

    try:
        # URLから画像データをダウンロード
        print("⏳ Downloading image...")
        image_response = requests.get(image_url)
        image_response.raise_for_status() # エラーチェック
        image_content = image_response.content
        print("✅ Image downloaded.")

        # Vertex AIに渡すための画像パートを作成
        image_part = Part.from_data(
            data=image_content,
            mime_type="image/png" # Discordの添付ファイルはPNGが多いと仮定
        )

        # プロンプトと画像をモデルに渡す
        prompt = "この画像について、写っているものを詳細に、客観的に説明してください。"
        print("⏳ Generating description...")
        response = multimodal_model.generate_content([image_part, prompt])
        print("✅ Description generated.")
        
        # テキスト部分だけを返す
        return jsonify({'text': response.text})

    except requests.exceptions.RequestException as e:
        print(f"❌ Error downloading image: {e}")
        return jsonify({'error': f"Failed to download image from URL: {e}"}), 500
    except Exception as e:
        print(f"❌ Error describing image: {e}")
        return jsonify({'error': str(e)}), 500


# 画像を配信するためのエンドポイント
@app.route('/images/<filename>')
def get_image(filename):
    return flask.send_from_directory(IMG_DIR, filename)

# --- サーバーの起動 ---
if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)
