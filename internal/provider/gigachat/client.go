package gigachat

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

const (
	OAuthUrl           = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	ModelsUrl          = "https://gigachat.devices.sberbank.ru/api/v1/models"
	GetCompletionstUrl = "https://gigachat.devices.sberbank.ru/api/v1/chat/completions"
)

type Token struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

type Client struct {
	cli     *fasthttp.Client
	authKey string
	token   string
	expire  time.Time
}

type CompletionResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromtTokens      int `json:"promt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
	Object string `json:"object"`
}

func NewClient(authKey string) *Client {
	cli := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, time.Duration(30)*time.Second)
		},
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Client{cli: cli, authKey: authKey}
}

func (c *Client) GetToken() error {
	if !c.expire.IsZero() && c.expire.After(time.Now()) {
		return nil
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(OAuthUrl)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Add("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+c.authKey)
	req.Header.Set("RqUID", uuid.New().String())
	req.SetBodyString("scope=GIGACHAT_API_PERS")

	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := c.cli.DoTimeout(req, resp, time.Duration(10)*time.Second); err != nil {
		return fmt.Errorf("timeout: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("status code: %v %v", resp.StatusCode(), resp.Body())
	}

	var token Token
	if err := json.Unmarshal(resp.Body(), &token); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w %v", err, resp.Body())
	}

	c.token = token.AccessToken
	c.expire = time.Unix(token.ExpiresAt, 0)

	return nil
}

func (c *Client) GetModels() string {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(ModelsUrl)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.Header.Add("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := c.cli.DoTimeout(req, resp, time.Duration(10)*time.Second); err != nil {
		fmt.Printf("Timeout, error: %v\n", err)
		return ""
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		fmt.Printf("Status code: %v\n", resp.StatusCode())
		return ""
	}

	return string(resp.Body())
}

func (c *Client) GetCompletions(text string) (string, error) {
	payload := map[string]any{
		"model": "GigaChat",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": text,
			},
		},
		"temperature":        1,
		"top_p":              0.1,
		"n":                  1,
		"stream":             false,
		"max_tokens":         512,
		"repetition_penalty": 1,
		"update_interval":    0,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return "", fmt.Errorf("marshal error: %v", err)
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(GetCompletionstUrl)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.Add("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.SetBody(body)

	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := c.cli.DoTimeout(req, resp, time.Duration(10)*time.Second); err != nil {
		return "", fmt.Errorf("timeout, error: %v", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return "", fmt.Errorf("status code: %v", resp.StatusCode())
	}

	var res CompletionResponse

	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(res.Choices) == 0 {
		return "", fmt.Errorf("empty choices")
	}

	return res.Choices[0].Message.Content, nil
}
