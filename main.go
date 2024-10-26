package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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
	// Декодируем изображение, чтобы получить его размеры
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("Не удалось декодировать изображение %s: %v", name, err)
		return ""
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	// log.Printf("Размер изображения %v and %v", width, height)
	if width < 60 && height < 60 {
		log.Printf("Изображение %s слишком маленькое (%dx%d), пропускаем", name, width, height)
		return ""
	}

	fileName := filepath.Base(name)
	filePath := filepath.Join("images", fileName)
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		log.Printf("Ошибка сохранения медиафайла %s: %v", name, err)
		return ""
	}
	return filePath
}

// Измененная функция, возвращающая флаг наличия контекста
func FindRelevantContext(question string, elements []DocumentElement) (string, []string, bool) {
	keywords := FindKeywords(question)
	var contextText strings.Builder
	var imagePaths []string
	found := false

	for i, element := range elements {
		if element.Type == "text" {
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(element.Content), keyword) {
					found = true
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

	return contextText.String(), imagePaths, found
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
			{Role: "system", Text: "Ты — умный ассистент, помогающий пользователям работать с приложением. Отвечай только на вопросы, связанные с документацией. Если вопрос не относится к документации, ответь: \"Такой информации нет, вы можете обратиться к разработчику.\""},
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

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
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

	// Парсинг документации
	elements, err := ParseDocument("./train_data_Sila/data.docx")
	if err != nil {
		log.Fatalf("Ошибка парсинга документа: %v", err)
	}

	router := gin.Default()
	router.Use(CORSMiddleware())

	// Обслуживание статических файлов (изображений)
	router.Static("/images", "./images")

	// Создаем файл для хранения вопросов и ответов
	qaFilePath := "qa_data.json"

	router.POST("/ask", func(c *gin.Context) {
		var request struct {
			Question string `json:"question" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userText := request.Question

		// Проверка на некорректный или бессмысленный вопрос
		if isInvalidQuestion(userText) {
			c.JSON(http.StatusOK, gin.H{
				"answer": "Вопрос некорректный. Пожалуйста, уточните свой вопрос.",
				"images": []string{},
			})
			return
		}

		contextText, imagePaths, found := FindRelevantContext(userText, elements)

		if !found {
			// Если контекст не найден, отвечаем, что информации нет
			c.JSON(http.StatusOK, gin.H{
				"answer": "Такой информации нет, вы можете обратиться к разработчику.",
				"images": []string{},
			})
			return
		}

		response, err := GenerateCompletion(userText, contextText, iamToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Генерация URL для изображений
		imageSet := make(map[string]struct{})
		for _, path := range imagePaths {
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			imageURL := fmt.Sprintf("%s://%s/images/%s", scheme, c.Request.Host, filepath.Base(path))
			imageSet[imageURL] = struct{}{}
		}

		// Преобразуем карту обратно в массив
		imageURLs := make([]string, 0, len(imageSet))
		for url := range imageSet {
			imageURLs = append(imageURLs, url)
		}
		log.Printf("Generated image URLs: %v", imageURLs)

		// Сохранение вопроса и ответа в JSON
		qa := QA{
			ID:       len(imageURLs), // Просто пример ID
			Question: userText,
			Answer:   response,
			Images:   imageURLs,
			Date:     time.Now().Format(time.RFC3339),
		}

		if err := saveQA(qaFilePath, qa); err != nil {
			log.Printf("Ошибка сохранения вопроса и ответа: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"answer": response,
			"images": imageURLs,
		})
	})

	router.GET("/history", func(c *gin.Context) {
		var qas []QA

		data, err := ioutil.ReadFile(qaFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при чтении файла истории"})
			return
		}

		if err := json.Unmarshal(data, &qas); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при разборе JSON"})
			return
		}

		c.JSON(http.StatusOK, qas)
	})

	router.POST("/deleteqa", ClearJSONFileHandler)

	router.Run(":8080")
}

// Функция для проверки корректности вопроса
func isInvalidQuestion(question string) bool {
	trimmed := strings.TrimSpace(question)
	if len(trimmed) < 5 {
		return true
	}

	// Проверка на наличие недопустимых символов
	invalidChars := regexp.MustCompile(`[^\w\s\p{L}\p{N}\p{P}]`)
	if invalidChars.MatchString(trimmed) {
		return true
	}

	return false
}

// хранения истории в json
type QA struct {
	ID       int      `json:"id"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Images   []string `json:"images"` // Список ссылок на изображения
	Date     string   `json:"date"`   // Можно использовать time.Time, но проще хранить как строку
}

func saveQA(filePath string, qa QA) error {
	var qas []QA

	// Чтение существующего файла
	if _, err := os.Stat(filePath); err == nil {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		json.Unmarshal(data, &qas)
	}

	// Добавление нового вопроса и ответа
	qas = append(qas, qa)

	// Сохранение в файл
	data, err := json.MarshalIndent(qas, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// func getQAs(filePath string) ([]QA, error) {
// 	var qas []QA

// 	// Чтение файла
// 	data, err := ioutil.ReadFile(filePath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Десериализация JSON
// 	if err := json.Unmarshal(data, &qas); err != nil {
// 		return nil, err
// 	}

// 	return qas, nil
// }

// очистка файла QA
func ClearJSONFile(filepath string) error {
	emptyJSON := make(map[string]interface{})
	file, err := os.Create(filepath) // Открываем файл на запись, перезаписывая его содержимое
	if err != nil {
		return err
	}
	defer file.Close()

	// Записываем пустой JSON-объект в файл
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Опционально: форматируем с отступами
	if err := encoder.Encode(emptyJSON); err != nil {
		return err
	}

	return nil
}

func ClearJSONFileHandler(c *gin.Context) {
	filePath := "./qa_data.json"

	// Вызываем функцию очистки
	if err := ClearJSONFile(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при очистке JSON-файла",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "JSON-файл успешно очищен",
	})
}
