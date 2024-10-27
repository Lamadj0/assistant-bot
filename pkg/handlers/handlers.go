package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"assistant-bot/pkg/ai"
	"assistant-bot/pkg/models"
	"assistant-bot/pkg/storage"
	"assistant-bot/pkg/utils"

	"github.com/gin-gonic/gin"
)

var (
	elements   []models.DocumentElement
	iamToken   string
	qaFilePath = "qa_data.json"
)

func InitHandlers(router *gin.Engine, docElements []models.DocumentElement, token string) {
	elements = docElements
	iamToken = token

	router.POST("/ask", askHandler)
	router.GET("/history", historyHandler)
	router.POST("/deleteqa", deleteQAHandler)
}

func askHandler(c *gin.Context) {
	var request struct {
		Question string `json:"question" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userText := request.Question

	// Проверка на некорректный или бессмысленный вопрос
	if utils.IsInvalidQuestion(userText) {
		// Создание объекта для сохранения некорректного запроса
		invalidQA := models.QA{
			ID:       -1,
			Question: userText,
			Answer:   "Вопрос некорректный. Пожалуйста, уточните свой вопрос.",
			Images:   []string{},
			Date:     time.Now().Format(time.RFC3339),
		}

		if err := storage.SaveQA(qaFilePath, invalidQA); err != nil {
			log.Printf("Ошибка сохранения некорректного вопроса и ответа: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"answer": "Вопрос некорректный. Пожалуйста, уточните свой вопрос.",
			"images": []string{},
		})
		return
	}

	contextText, imagePaths, found := utils.FindRelevantContext(userText, elements)

	if !found {
		// Создание объекта для сохранения запроса без найденного контекста
		notFoundQA := models.QA{
			ID:       -1,
			Question: userText,
			Answer:   "Такой информации нет, вы можете обратиться к разработчику.",
			Images:   []string{},
			Date:     time.Now().Format(time.RFC3339),
		}

		if err := storage.SaveQA(qaFilePath, notFoundQA); err != nil {
			log.Printf("Ошибка сохранения запроса без контекста: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"answer": "Такой информации нет, вы можете обратиться к разработчику.",
			"images": []string{},
		})
		return
	}

	response, err := ai.GenerateCompletion(userText, contextText, iamToken)
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

	// Сохранение корректного вопроса и ответа в JSON
	qa := models.QA{
		ID:       len(imageURLs), // Просто пример ID
		Question: userText,
		Answer:   response,
		Images:   imageURLs,
		Date:     time.Now().Format(time.RFC3339),
	}

	if err := storage.SaveQA(qaFilePath, qa); err != nil {
		log.Printf("Ошибка сохранения вопроса и ответа: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"answer": response,
		"images": imageURLs,
	})
}

func historyHandler(c *gin.Context) {
	qas, err := storage.LoadQA(qaFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при чтении файла истории"})
		return
	}

	c.JSON(http.StatusOK, qas)
}

func deleteQAHandler(c *gin.Context) {
	filePath := qaFilePath

	// Вызываем функцию очистки
	if err := storage.ClearJSONFile(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка при очистке JSON-файла",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "JSON-файл успешно очищен",
	})
}
