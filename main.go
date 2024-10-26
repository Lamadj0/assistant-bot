// main.go

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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

// Функция для парсинга .docx файла и извлечения текста и изображений
func ParseDocument(filePath string) ([]DocumentElement, error) {
	var elements []DocumentElement

	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var documentXML []byte
	mediaFiles := make(map[string]string)

	// Создаем директорию для изображений, если ее нет
	if _, err := os.Stat("images"); os.IsNotExist(err) {
		os.Mkdir("images", os.ModePerm)
	}

	for _, file := range reader.File {
		switch {
		case file.Name == "word/document.xml":
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			documentXML, err = ioutil.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
		case strings.HasPrefix(file.Name, "word/media/"):
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			mediaData, err := ioutil.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			mediaFiles[file.Name] = SaveMediaFile(file.Name, mediaData)
		}
	}

	if documentXML == nil {
		return nil, fmt.Errorf("не удалось найти document.xml")
	}

	// Регулярные выражения для поиска текстовых элементов и изображений
	paragraphRegex := regexp.MustCompile(`(?s)<w:p[^>]*>.*?</w:p>`)
	textRegex := regexp.MustCompile(`(?s)<w:t[^>]*>(.*?)</w:t>`)
	imageRegex := regexp.MustCompile(`<a:blip[^>]+r:embed="([^"]+)"`)

	paragraphs := paragraphRegex.FindAll(documentXML, -1)
	if paragraphs == nil {
		return nil, fmt.Errorf("не удалось найти параграфы в документе")
	}

	// Считываем отношения из word/_rels/document.xml.rels
	relationships := make(map[string]string)
	for _, file := range reader.File {
		if file.Name == "word/_rels/document.xml.rels" {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			relsData, err := ioutil.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
			relRegex := regexp.MustCompile(`<Relationship[^>]+Id="([^"]+)"[^>]+Target="([^"]+)"`)
			relMatches := relRegex.FindAllSubmatch(relsData, -1)
			for _, match := range relMatches {
				id := string(match[1])
				target := string(match[2])
				relationships[id] = target
			}
		}
	}

	for _, p := range paragraphs {
		// Ищем текст в параграфе
		texts := textRegex.FindAllSubmatch(p, -1)
		var paragraphText string
		for _, t := range texts {
			paragraphText += string(t[1])
		}
		if paragraphText != "" {
			elements = append(elements, DocumentElement{
				Type:    "text",
				Content: paragraphText,
			})
		}

		// Ищем изображения в параграфе
		images := imageRegex.FindAllSubmatch(p, -1)
		for _, img := range images {
			rid := string(img[1])
			// Получаем путь к файлу изображения из отношений
			target, ok := relationships[rid]
			if !ok {
				continue
			}
			imagePath := ""
			for mediaPath, savedPath := range mediaFiles {
				if strings.HasSuffix(mediaPath, target) {
					imagePath = savedPath
					break
				}
			}
			if imagePath != "" {
				elements = append(elements, DocumentElement{
					Type:    "image",
					Content: imagePath,
				})
			}
		}
	}

	return elements, nil
}

// Функция для сохранения медиафайла и возврата пути к нему
func SaveMediaFile(name string, data []byte) string {
	fileName := filepath.Base(name)
	filePath := filepath.Join("images", fileName)
	err := ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		log.Printf("Ошибка сохранения медиафайла %s: %v", name, err)
		return ""
	}
	return filePath
}

func FindRelevantContext(question string, elements []DocumentElement) (string, []string) {
	keywords := FindKeywords(question)
	var contextText strings.Builder
	var imagePaths []string

	for i, element := range elements {
		if element.Type == "text" {
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(element.Content), keyword) {
					// Добавляем текст в контекст
					contextText.WriteString(element.Content)
					contextText.WriteString("\n")

					// Проверяем соседние элементы на наличие изображений
					if i+1 < len(elements) && elements[i+1].Type == "image" {
						imagePaths = append(imagePaths, elements[i+1].Content)
					}
					if i > 0 && elements[i-1].Type == "image" {
						imagePaths = append(imagePaths, elements[i-1].Content)
					}
					break
				}
			}
		}
	}

	if contextText.Len() == 0 {
		return "Контекст не найден.", nil
	}

	return contextText.String(), imagePaths
}

func FindKeywords(text string) []string {
	stopWords := map[string]bool{
		"и": true, "в": true, "на": true, "с": true, "по": true, "для": true,
		// Добавьте другие стоп-слова по необходимости
	}
	words := strings.Fields(text)
	keywordMap := make(map[string]struct{})

	for _, word := range words {
		cleanedWord := strings.ToLower(word)
		cleanedWord = strings.Trim(cleanedWord, ".,!?\"'") // Убираем знаки препинания
		if len(cleanedWord) > 3 && !stopWords[cleanedWord] {
			keywordMap[cleanedWord] = struct{}{}
		}
	}

	keywords := make([]string, 0, len(keywordMap))
	for keyword := range keywordMap {
		keywords = append(keywords, keyword)
	}
	return keywords
}

func GenerateCompletion(userText, context, iamToken string) (string, error) {
	url := "https://llm.api.cloud.yandex.net/foundationModels/v1/completion"

	requestData := RequestData{
		ModelUri: "gpt://b1gjp5vama10h4due384/yandexgpt/latest",
		CompletionOptions: CompletionOptions{
			Stream:      false,
			Temperature: 0.6,
			MaxTokens:   2000,
		},
		Messages: []Message{
			{Role: "system", Text: "Ты — умный ассистент, помогающий пользователям работать с приложением. Обрати внимание, что в документации могут быть изображения, связанные с текстом."},
			{Role: "user", Text: fmt.Sprintf("Документация:\n%s", context)},
			{Role: "user", Text: fmt.Sprintf("Вопрос пользователя: %s", userText)},
		},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", iamToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("неудачный запрос: %s, %s", resp.Status, string(bodyBytes))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var responseData ResponseData
	if err := json.Unmarshal(body, &responseData); err != nil {
		return "", err
	}

	if len(responseData.Result.Alternatives) > 0 {
		return responseData.Result.Alternatives[0].Message.Text, nil
	}

	return "Ответ не получен", nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки файла .env")
	}

	// Получаем значение переменной API_KEY
	iamToken := os.Getenv("API_KEY")
	if iamToken == "" {
		log.Fatal("API_KEY не установлен")
	}

	// Остальной код программы...
	log.Println("Токен:", iamToken)

	// Парсинг документации
	elements, err := ParseDocument("./train_data_Sila/data.docx")
	if err != nil {
		log.Fatalf("Ошибка парсинга документа: %v", err)
	}

	router := gin.Default()

	// Обслуживание статических файлов (изображений)
	router.Static("/images", "./images")

	router.POST("/ask", func(c *gin.Context) {
		var request struct {
			Question string `json:"question" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userText := request.Question
		contextText, imagePaths := FindRelevantContext(userText, elements)

		response, err := GenerateCompletion(userText, contextText, iamToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Генерация URL для изображений
		imageURLs := make([]string, len(imagePaths))

		for i, path := range imagePaths {
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			imageURLs[i] = fmt.Sprintf("%s://%s/images/%s", scheme, c.Request.Host, filepath.Base(path))
		}
		log.Printf("Generated image URLs: %v", imageURLs)

		c.JSON(http.StatusOK, gin.H{
			"answer": response,
			"images": imageURLs,
		})
	})

	router.Run(":8080")
}
