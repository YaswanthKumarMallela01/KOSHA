package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/YaswanthKumarMallela01/kosha/internal/markup"
)

type Client struct {
	apiKey     string
	model      string
	timeout    time.Duration
	httpClient *http.Client
}

func NewClient(apiKey, modelName string) *Client {
	timeout := 30 * time.Second
	return &Client{
		apiKey:     apiKey,
		model:      modelName,
		timeout:    timeout,
		httpClient: &http.Client{Timeout: timeout},
	}
}

type BlockRequest struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type BlockResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type geminiRequest struct {
	SystemInstruction systemInstruction `json:"system_instruction"`
	Contents          []content         `json:"contents"`
	GenerationConfig  generationConfig  `json:"generationConfig"`
}

type systemInstruction struct {
	Parts []part `json:"parts"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMimeType string         `json:"responseMimeType"`
	ResponseSchema   responseSchema `json:"responseSchema"`
}

type responseSchema struct {
	Type       string                    `json:"type"`
	Items      *responseSchemaItems      `json:"items,omitempty"`
}

type responseSchemaItems struct {
	Type       string                    `json:"type"`
	Properties map[string]responseSchema `json:"properties"`
	Required   []string                  `json:"required"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (c *Client) RefineBlocks(ctx context.Context, blocks []BlockRequest) ([]BlockResponse, error) {
	blocksJSON, err := json.Marshal(blocks)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal blocks: %w", err)
	}

	promptText := fmt.Sprintf("%s\n\nJSON array of text blocks to refine:\n%s", SystemInstruction, string(blocksJSON))

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": promptText},
				},
			},
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	cleanModel := strings.TrimPrefix(c.model, "models/")
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", cleanModel, c.apiKey)

	var resp *http.Response
	var lastErr error
	for retries := 0; retries < 3; retries++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to create http request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err = c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(1<<retries) * time.Second)
			continue
		}

		if (resp.StatusCode == 429 || resp.StatusCode == 503) && retries < 2 {
			resp.Body.Close()
			select {
			case <-time.After(time.Duration(1<<retries) * time.Second):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		break
	}

	if resp == nil {
		return nil, fmt.Errorf("http request failed after retries: %w", lastErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty response from model")
	}

	respText := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	// Strip markdown code fences if model wrapped response in ```json ... ```
	if strings.HasPrefix(respText, "```") {
		lines := strings.Split(respText, "\n")
		if len(lines) >= 2 {
			if strings.HasPrefix(lines[len(lines)-1], "```") {
				lines = lines[1 : len(lines)-1]
			} else {
				lines = lines[1:]
			}
			respText = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}

	var blockResponses []BlockResponse
	if err := json.Unmarshal([]byte(respText), &blockResponses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response from model: %w", err)
	}

	return blockResponses, nil
}

func ValidateResponse(requests []BlockRequest, responses []BlockResponse) error {
	if len(requests) != len(responses) {
		return fmt.Errorf("length mismatch: expected %d, got %d", len(requests), len(responses))
	}

	reqIDMap := make(map[string]bool)
	for _, req := range requests {
		reqIDMap[req.ID] = true
	}

	respIDMap := make(map[string]bool)
	for _, resp := range responses {
		if !reqIDMap[resp.ID] {
			return fmt.Errorf("unexpected ID in response: %s", resp.ID)
		}
		if resp.Text == "" {
			return fmt.Errorf("empty text field for ID: %s", resp.ID)
		}
		respIDMap[resp.ID] = true
	}

	for id := range reqIDMap {
		if !respIDMap[id] {
			return fmt.Errorf("missing ID in response: %s", id)
		}
	}

	return nil
}

func CheckHighlightDensity(text string) (bool, float64) {
	re := regexp.MustCompile(`(==.*?==|\^\^.*?\^\^)`)
	matches := re.FindAllString(text, -1)
	hlCount := 0
	for _, m := range matches {
		hlCount += len(m) - 4
	}

	totalLen := len(text)
	if totalLen == 0 {
		return false, 0
	}

	density := float64(hlCount) / float64(totalLen)
	return density > 0.15, density
}

func SanitizeHighlights(text string, maxDensity float64) string {
	exceeds, _ := CheckHighlightDensity(text)
	if !exceeds {
		return text
	}

	re := regexp.MustCompile(`(==.*?==|\^\^.*?\^\^)`)

	for {
		matches := re.FindAllStringIndex(text, -1)
		if len(matches) == 0 {
			break
		}

		lastMatch := matches[len(matches)-1]
		start, end := lastMatch[0], lastMatch[1]
		
		mStr := text[start:end]
		content := mStr[2 : len(mStr)-2]
		
		text = text[:start] + content + text[end:]
		
		exceeds, density := CheckHighlightDensity(text)
		if !exceeds || density <= maxDensity {
			break
		}
	}

	return text
}

func ComputeEditDistance(original, refined string) float64 {
	origStripped := markup.StripMarkup(original)
	refStripped := markup.StripMarkup(refined)

	origWords := strings.Fields(origStripped)
	refWords := strings.Fields(refStripped)

	dist := levenshteinDistance(origWords, refWords)
	
	maxLen := len(origWords)
	if len(refWords) > maxLen {
		maxLen = len(refWords)
	}
	if maxLen == 0 {
		return 0
	}
	return float64(dist) / float64(maxLen)
}

func levenshteinDistance(a, b []string) int {
	la := len(a)
	lb := len(b)
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
	}
	for i := 0; i <= la; i++ {
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			min := d[i-1][j] + 1
			if d[i][j-1]+1 < min {
				min = d[i][j-1] + 1
			}
			if d[i-1][j-1]+cost < min {
				min = d[i-1][j-1] + cost
			}
			d[i][j] = min
		}
	}
	return d[la][lb]
}

type DiffBlock struct {
	ID             string
	Original       string
	Refined        string
	EditDistance   float64
	HeavilyChanged bool 
	Accepted       bool
	Rejected       bool
	KeepAsIs       bool 
}

func PrepareDiff(requests []BlockRequest, responses []BlockResponse) []DiffBlock {
	respMap := make(map[string]string)
	for _, r := range responses {
		respMap[r.ID] = r.Text
	}

	var diffs []DiffBlock
	for _, req := range requests {
		refText := respMap[req.ID]
		dist := ComputeEditDistance(req.Text, refText)
		diffs = append(diffs, DiffBlock{
			ID:             req.ID,
			Original:       req.Text,
			Refined:        refText,
			EditDistance:   dist,
			HeavilyChanged: dist > 0.35,
			Accepted:       false,
			Rejected:       false,
			KeepAsIs:       false,
		})
	}
	return diffs
}
