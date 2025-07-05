using Tesseract.Net.SDK;
using Tesseract.Net.SDK.Model;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

app.MapPost("/read-text", async (HttpRequest request) =>
{
    try
    {
        if (!request.HasFormContentType)
        {
            return Results.BadRequest("Invalid content type. Please use multipart/form-data.");
        }

        var form = await request.ReadFormAsync();
        var file = form.Files.GetFile("image");

        if (file == null)
        {
            return Results.BadRequest("Image file is required.");
        }

        Console.WriteLine("✅ C# Server: Image received. Starting OCR process...");

        // 画像データをメモリに読み込む
        await using var memoryStream = new MemoryStream();
        await file.CopyToAsync(memoryStream);
        var imageBytes = memoryStream.ToArray();
        
        // Tesseractエンジンを初期化 (日本語と英語を読み取れるように設定)
        using var engine = new TesseractEngine(language: Language.Japanese | Language.English, modelPath: "./tessdata");
        
        // OCRを実行
        var result = engine.Process(imageBytes);
        
        Console.WriteLine($"✅ C# Server: OCR process completed. Text found: {result.Text.Length} chars.");

        // 読み取ったテキストをJSON形式で返す
        return Results.Ok(new { text = result.Text });
    }
    catch (Exception ex)
    {
        Console.WriteLine($"❌ C# Server Error: {ex.Message}");
        return Results.Problem("An error occurred during OCR processing.");
    }
});

// ポート5002でサーバーを起動
app.Run("http://localhost:5002");