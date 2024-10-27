package models

type CompletionOptions struct {
	Stream      bool    `json:"stream"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type RequestData struct {
	ModelUri          string            `json:"modelUri"`
	CompletionOptions CompletionOptions `json:"completionOptions"`
	Messages          []Message         `json:"messages"`
}

type ResponseData struct {
	Result struct {
		Alternatives []struct {
			Message struct {
				Role string `json:"role"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"alternatives"`
	} `json:"result"`
}

type DocumentElement struct {
	Type    string // "text" или "image"
	Content string // Текст или путь к изображению
}

type QA struct {
	ID       int      `json:"id"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Images   []string `json:"images"`
	Date     string   `json:"date"`
}
