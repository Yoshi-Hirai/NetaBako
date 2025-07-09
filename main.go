package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"NetaBako/fileio"

	"gopkg.in/yaml.v2"
)

const (
	projectID  = "neta-bako"                            // ← あなたのプロジェクトID
	location   = "us-central1"                          // ← あなたのロケーション（例: us-central1）
	modelID    = "gemini-2.5-pro"                       // ← 使用するモデルID（例: gemini-1.0-pro）
	apiBaseURL = "https://aiplatform.googleapis.com/v1" // Gemini APIのベースURL
	// 上記のURLは、実際のAPIエンドポイントに合わせて調整してください)
)

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// getAccessToken は、gcloud CLIを使ってGoogle Cloudのアクセストークンを取得します。
// このトークンは、Gemini APIへの認証に使用されます。
func getAccessToken() (string, error) {
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("トークン取得失敗: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// LoadPromptsYaml は、指定されたパスからYAMLファイルを読み込み、マップ形式で返します。
func LoadPromptsYaml(path string) (map[string]string, error) {
	data, err := fileio.FileIoRead(path)
	if err != nil {
		return nil, err
	}
	var result map[string]string
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// このコードは、Google CloudのGemini APIを使ってSNS投稿のネタを生成するサンプルです。
// 実行には、gcloud CLIがインストールされており、認証済みである必要があります。
func main() {
	// プロンプトの読み込み
	promptKey := flag.String("prompt", "", "使用するプロンプトのキーを指定します（例: X)")
	promptKeyShort := flag.String("p", "", "使用するプロンプトのキーを短縮形で指定します（例: X)")
	themeKey := flag.String("theme", "", "テーマを指定します（例: 旅行）")
	themeKeyShort := flag.String("t", "", "テーマを短縮形で指定します（例: 旅行）")
	flag.Parse()

	// YAMLファイルからプロンプトを読み込む
	prompts, err := LoadPromptsYaml("./prompts.yaml")
	if err != nil {
		fmt.Println("⚠️ プロンプトの読み込み失敗:", err)
		return
	}
	//fmt.Println("🔍 プロンプトテンプレート:", prompts)

	// プロンプトのキーを決定
	selectedKey := *promptKey
	if *promptKeyShort != "" {
		selectedKey = *promptKeyShort
	}
	if selectedKey == "" {
		fmt.Println("⚠️ プロンプトのキーが指定されていません。-prompt または -p オプションを使用してください。")
		return
	}
	// プロンプトキーの存在チェック
	template, ok := prompts[selectedKey]
	if !ok {
		fmt.Printf("⚠️ プロンプトキー '%s' が見つかりません。利用可能なキー: %v\n", selectedKey, prompts)
		return
	}

	// テーマの設定
	selectedTheme := *themeKey
	if *themeKeyShort != "" {
		selectedTheme = *themeKeyShort
	}
	if selectedTheme == "" {
		fmt.Println("⚠️ テーマが指定されていません。-theme または -t オプションを使用してください。")
		return
	}
	//fmt.Printf("✅ 選択されたプロンプト (%s):\n%s テーマ:%s\n", selectedKey, template, selectedTheme)

	// メッセージ（ここを書き換えれば他の質問もOK）
	//userInput := "こんにちは！今日のSNSに投稿したくなるようなネタを1つください。"
	userInput := strings.ReplaceAll(template, "{{THEME}}", selectedTheme)
	//fmt.Println("💬 ユーザー入力:", userInput)

	// Gemini APIのURL構築
	endpoint := fmt.Sprintf(
		"%s/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		apiBaseURL, projectID, location, modelID,
	)
	//fmt.Println("🔗 リクエストURL:", endpoint)

	// JSONボディ構築
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": userInput},
				},
			},
		},
	}
	jsonData, _ := json.Marshal(requestBody)

	// アクセストークン取得
	token, err := getAccessToken()
	if err != nil {
		panic(err)
	}
	//fmt.Println("🔑 アクセストークン取得成功:", token)

	// リクエスト送信
	//fmt.Println("📤 リクエスト送信中...", string(jsonData))
	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	//fmt.Printf("📡 HTTPステータスコード: %d\n", resp.StatusCode)

	// レスポンス読み取り
	body, _ := io.ReadAll(resp.Body)
	//fmt.Println("🪵 レスポンスボディ全文:")
	//fmt.Println(string(body))

	// JSONパース
	var result GeminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("⚠️ JSONパース失敗:", err)
		fmt.Println("📦 生データ:", string(body))
		return
	}

	// テキスト部分出力
	fmt.Println("🔻 Gemini 応答:")
	for _, candidate := range result.Candidates {
		for _, part := range candidate.Content.Parts {
			fmt.Println("👉", part.Text)
		}
	}
}
