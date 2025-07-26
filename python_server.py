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

# FlaskアプリとVertex AIの初期化
app = flask.Flask(__name__)
vertexai.init() 

# 画像生成
image_model = ImageGenerationModel.from_pretrained("imagen-4.0-ultra-generate-preview-06-06") 
# 多モーダル
multimodal_model = GenerativeModel("gemini-2.5-pro")

# APIエンドポイント
# 画像生成用エンドポイント
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

# テキスト生成用エンドポイント
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

# 用画像認識エンドポイント
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

        # Vertex AIに渡すための画像パート
        image_part = Part.from_data(
            data=image_content,
            mime_type="image/png"
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


# クイズ用エンドポイント
@app.route('/generate-quiz', methods=['POST'])
def generate_quiz():
    data = request.get_json()
    if not data or 'topic' not in data:
        return jsonify({'error': 'topic is required'}), 400

    topic = data.get('topic', 'ランダムなトピック')
    history = data.get('history', []) # 過去の質問リストを受け取る

    print(f"✅ Received Quiz request for topic: {topic}")
    print(f"📖 Received history with {len(history)} questions.")

    history_prompt = ""
    if history:
        history_prompt = "ただし、以下のリストにある質問絶対に出題しないでください。\n- " + "\n- ".join(history)

    prompt = f"""
「{topic}」に関する、ユニークで面白い4択クイズを1問生成してください。
あなたの応答は、必ず以下のJSON形式に従ってください。他のテキストは一切含めないでください。

{{
  "question": "ここに問題文",
  "options": [
    "選択肢A",
    "選択肢B",
    "選択肢C",
    "選択肢D"
  ],
  "correct_answer_index": 2, 
  "explanation": "ここに簡単な解説"
}}

{history_prompt}
"""

    try:
        print("⏳ Generating new quiz...")
        response = multimodal_model.generate_content(prompt)
        print("✅ Quiz generated.")
        
        # AIの出力からJSON部分だけを抽出する（念のため）
        json_text = response.text.strip()
        if json_text.startswith("```json"):
            json_text = json_text[7:-4].strip()

        # JSONとしてパースできるか検証
        import json
        json.loads(json_text) 

        return app.response_class(
            response=json_text,
            status=200,
            mimetype='application/json'
        )

    except Exception as e:
        print(f"❌ Error generating quiz: {e}")
        return jsonify({'error': str(e)}), 500


# ユーザー活動分析用エンドポイント
@app.route('/analyze-user-activity', methods=['POST'])
def analyze_user_activity():
    data = request.get_json()
    if not data or 'user_id' not in data or 'username' not in data or 'joined_at' not in data or 'roles' not in data:
        return jsonify({'error': 'user_id, username, joined_at, and roles are required'}), 400

    user_id = data['user_id']
    username = data['username']
    joined_at = data['joined_at']
    roles = data['roles']

    print(f"✅ Received User Activity Analysis request for {username} ({user_id})")

    # Geminiに渡すプロンプトを作成
    prompt = f"""
以下のDiscordユーザーの情報を元に、そのユーザーの活動傾向を簡潔に分析してください。
ユーザー名: {username}
参加日時: {joined_at}
ロール: {', '.join(roles) if roles else 'なし'}

分析は、ユーザーの一般的な活動傾向、サーバーへの貢献度、興味の可能性などに焦点を当ててください。
「このユーザーは...」という形式で始めてください。
"""

    try:
        print("⏳ Analyzing user activity...")
        response = multimodal_model.generate_content(prompt)
        print("✅ User activity analyzed.")
        return jsonify({'text': response.text})

    except Exception as e:
        print(f"❌ Error analyzing user activity: {e}")
        return jsonify({'error': str(e)}), 500


# 画像配信エンドポイント
@app.route('/images/<filename>')
def get_image(filename):
    return flask.send_from_directory(IMG_DIR, filename)

# サーバー起動
if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)
