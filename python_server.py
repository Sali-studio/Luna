import os
import flask
from flask import request, jsonify, Response
import vertexai
from vertexai.preview.vision_models import ImageGenerationModel
from vertexai.generative_models import GenerativeModel, Part
import random

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
    negative_prompt = data.get('negative_prompt', None) # ネガティブプロンプトを取得

    print(f"✅ Received Image prompt: {prompt}")
    if negative_prompt:
        print(f"🚫 Received Negative Prompt: {negative_prompt}")

    try:
        print("⏳ Generating image...")
        # モデルにネガティブプロンプトを渡す
        images = image_model.generate_images(
            prompt=prompt,
            negative_prompt=negative_prompt,
            number_of_images=1
        )
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

# テキスト生成用エンドポイント(ストリーミング)
@app.route('/generate-text-stream', methods=['POST'])
def generate_text_stream():
    data = request.get_json()
    if not data or 'prompt' not in data:
        return jsonify({'error': 'prompt is required'}), 400

    prompt = data['prompt']
    print(f"✅ Received Text Stream prompt: {prompt}")

    def generate():
        try:
            print("⏳ Generating text stream...")
            # stream=Trueを指定して、レスポンスをストリームで受け取る
            responses = multimodal_model.generate_content(prompt, stream=True)
            print("✅ Text stream started.")
            for response in responses:
                # 各チャンクをそのままクライアントに送信
                yield response.text
        except Exception as e:
            print(f"❌ Error generating text stream: {e}")
            # エラーが発生した場合も、ストリームを閉じるために空のデータを送信
            yield ""

    # ストリーミングレスポンスを返す
    return Response(generate(), mimetype='text/plain')

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

# OCR用エンドポイント
@app.route('/ocr', methods=['POST'])
def ocr():
    data = request.get_json()
    if not data or 'image_url' not in data:
        return jsonify({'error': 'image_url is required'}), 400

    image_url = data['image_url']
    print(f"✅ Received OCR Request for URL: {image_url}")

    try:
        # URLから画像データをダウンロード
        print("⏳ Downloading image for OCR...")
        image_response = requests.get(image_url)
        image_response.raise_for_status() # エラーチェック
        image_content = image_response.content
        print("✅ Image downloaded for OCR.")

        # Vertex AIに渡すための画像パート
        image_part = Part.from_data(
            data=image_content,
            mime_type="image/png" # MIMEタイプは適宜変更してください
        )

        # OCRに特化したプロンプト
        prompt = "この画像に含まれているすべてのテキストを、一字一句正確に書き出してください。他の説明や前置きは一切不要です。"
        print("⏳ Performing OCR...")
        response = multimodal_model.generate_content([image_part, prompt])
        print("✅ OCR completed.")
        
        # テキスト部分だけを返す
        return jsonify({'text': response.text})

    except requests.exceptions.RequestException as e:
        print(f"❌ Error downloading image for OCR: {e}")
        return jsonify({'error': f"Failed to download image from URL: {e}"}), 500
    except Exception as e:
        print(f"❌ Error during OCR: {e}")
        return jsonify({'error': str(e)}), 500


# クイズ用エンドポイント
@app.route('/generate-quiz', methods=['POST'])
def generate_quiz():
    data = request.get_json()
    if not data or 'topic' not in data:
        return jsonify({'error': 'topic is required'}), 400

    topic = data.get('topic', 'ランダムなトピック')
    history = data.get('history', [])

    print(f"✅ Received Quiz request for topic: {topic}")
    print(f"📖 Received history with {len(history)} questions.")

    history_prompt = ""
    if history:
        history_prompt = "ただし、以下のリストにある質問は絶対に出題しないでください。\n- " + "\n- ".join(history)

    prompt = f'''「{topic}」に関する、ユニークで面白い4択クイズを1問生成してください。
あなたの応答は、必ず以下のJSON形式に従ってください。他のテキストは一切含めないでください。

{{
  "question": "ここに問題文",
  "options": [
    "選択肢A",
    "選択肢B",
    "選択肢C",
    "正解の選択肢D"
  ],
  "correct_answer": "正解の選択肢D",
  "explanation": "ここに簡単な解説"
}}

{history_prompt}
'''

    try:
        print("⏳ Generating new quiz...")
        response = multimodal_model.generate_content(prompt)
        print("✅ Quiz generated.")
        
        json_text = response.text.strip()
        if json_text.startswith("```json"):
            json_text = json_text[7:-4].strip()

        import json
        quiz_data = json.loads(json_text)

        # --- Shuffle options and find the new correct index ---
        correct_answer_str = quiz_data['correct_answer']
        options = quiz_data['options']
        
        # Ensure the correct answer is actually in the options list
        if correct_answer_str not in options:
            # If not, something is wrong with the AI's output. Add it to be safe.
            options.append(correct_answer_str)

        random.shuffle(options)
        
        new_correct_index = -1
        for i, option in enumerate(options):
            if option == correct_answer_str:
                new_correct_index = i
                break
        
        if new_correct_index == -1:
            # This should not happen if the logic above is correct
            raise ValueError("Correct answer not found after shuffling")

        # Build the final JSON response
        final_quiz = {
            "question": quiz_data['question'],
            "options": options,
            "correct_answer_index": new_correct_index,
            "explanation": quiz_data['explanation']
        }

        return jsonify(final_quiz)

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
    prompt = f'''
以下のDiscordユーザーの情報を元に、そのユーザーの活動傾向を簡潔に分析してください。
ユーザー名: {username}
参加日時: {joined_at}
ロール: {', '.join(roles) if roles else 'なし'}

分析は、ユーザーの一般的な活動傾向、サーバーへの貢献度、興味の可能性などに焦点を当ててください。
「このユーザーは...」という形式で始めてください。
'''

    try:
        print("⏳ Analyzing user activity...")
        response = multimodal_model.generate_content(prompt)
        print("✅ User activity analyzed.")
        return jsonify({'text': response.text})

    except Exception as e:
        print(f"❌ Error analyzing user activity: {e}")
        return jsonify({'error': str(e)}), 500

# 新しいプロフィール分析エンドポイント
@app.route('/analyze-profile', methods=['POST'])
def analyze_profile():
    data = request.get_json()
    if not data or 'username' not in data:
        return jsonify({'error': 'username is required'}), 400

    username = data['username']
    roles = data.get('roles', [])
    recent_messages = data.get('recent_messages', [])

    print(f"✅ Received Profile Analysis request for {username}")

    # メッセージ履歴を整形
    message_history = "\n".join(recent_messages) if recent_messages else "まだ発言がありません。"

    # Geminiに渡すプロンプトを作成
    prompt = f'''
あなたはプロのプロファイラーです。
以下のDiscordユーザーの情報を元に、そのユーザーの人物像を創造的かつ洞察に満ちた文章で分析してください。
分析結果は、本人に直接見せることを想定し、ポジティブで面白い内容にしてください。

# ユーザー情報
- ユーザー名: {username}

# 最近の発言 (直近最大100件)
{message_history}

# 分析のポイント
- **話し方の特徴:** 丁寧、フレンドリー、絵文字をよく使うなど。
- **興味・関心:** 最近の会話から、何に興味があるように見えるか。
- **サーバー内での役割:** ムードメーカー、情報通、特定の話題の専門家など、どのような役割を担っているように見えるか。
- **ユニークなキャッチコピー:** 全てを総合して、その人に面白いキャッチコピーを付けてください。

分析結果は、以下のフォーマットで、マークダウンを使って記述してください。

### 🗣️ 話し方の特徴
ここに分析結果を記述。

### 興味・関心
ここに分析結果を記述。

### サーバー内での役割
ここに分析結果を記述。

### ✨ キャッチコピー
**ここにキャッチコピーを記述**
'''

    try:
        print("⏳ Analyzing profile...")
        response = multimodal_model.generate_content(prompt)
        print("✅ Profile analyzed.")
        return jsonify({'text': response.text})

    except Exception as e:
        print(f"❌ Error analyzing profile: {e}")
        return jsonify({'error': str(e)}), 500


# 画像配信エンドポイント
@app.route('/images/<filename>')
def get_image(filename):
    return flask.send_from_directory(IMG_DIR, filename)

# サーバー起動
if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001)