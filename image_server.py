# image_server.py
import os
import flask
from flask import request, jsonify
import vertexai
from vertexai.preview.vision_models import ImageGenerationModel
import time

IMG_DIR = "generated_images"
if not os.path.exists(IMG_DIR):
    os.makedirs(IMG_DIR)

# --- FlaskアプリとVertex AIの初期化 ---
app = flask.Flask(__name__)
vertexai.init()

model = ImageGenerationModel.from_pretrained("imagegeneration@006")

# --- APIエンドポイント ---
@app.route('/generate', methods=['POST'])
def generate_image():
    data = request.get_json()
    if not data or 'prompt' not in data:
        return jsonify({'error': 'prompt is required'}), 400

    prompt = data['prompt']
    print(f"✅ Received prompt: {prompt}")

    try:
        print("⏳ Generating image...")
        images = model.generate_images(prompt=prompt, number_of_images=1)

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
        print(f"❌ Error: {e}")
        return jsonify({'error': str(e)}), 500

@app.route('/images/<filename>')
def get_image(filename):
    return flask.send_from_directory(IMG_DIR, filename)

# --- サーバーの起動 ---
if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)