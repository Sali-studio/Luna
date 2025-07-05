using Genie, Images, Plots
using Genie.Renderer.Json

# マンデルブロ集合を計算する関数
function mandelbrot(c; max_iter=255)
    z = c
    for n = 1:max_iter
        if abs(z) > 2
            return n
        end
        z = z^2 + c
    end
    return max_iter
end

# 画像を生成するメインの関数
function generate_mandelbrot_image(width, height, x_min, x_max, y_min, y_max)
    img = Matrix{RGB{Float64}}(undef, height, width)
    for (i, y) in enumerate(range(y_max, stop=y_min, length=height))
        for (j, x) in enumerate(range(x_min, stop=x_max, length=width))
            c = x + y*im
            m = mandelbrot(c)
            
            # 計算結果を色にマッピング
            hue = 255 - m
            color_val = HSV(hue, 1.0, m < 255 ? 1.0 : 0.0)
            img[i, j] = convert(RGB, color_val)
        end
    end
    return img
end

# --- Webサーバーの定義 ---
d
# /mandelbrot というURLでリクエストを受け付ける
route("/mandelbrot") do
  println("✅ Request received to generate Mandelbrot set.")
  
  # 画像を生成
  img = generate_mandelbrot_image(800, 600, -2.0, 1.0, -1.0, 1.0)
  
  # 生成した画像を一時ファイルとして保存
  filepath = "mandelbrot_temp.png"
  save(filepath, img)
  println("✅ Image saved to: ", filepath)

  # 画像ファイルをそのまま返す
  Genie.Renderer.file(filepath, "image/png")
end

println("Julia Mandelbrot server is starting...")
# ポート8001でサーバーを起動
up(8001, "0.0.0.0", async=false)