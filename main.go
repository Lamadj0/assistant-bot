package main

import (
	"log"
	"os"

	"assistant-bot/pkg/docparser"
	"assistant-bot/pkg/handlers"
	"assistant-bot/pkg/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки файла .env")
	}

	iamToken := os.Getenv("API_KEY")
	if iamToken == "" {
		log.Fatal("API_KEY не установлен")
	}

	// Парсинг документации
	elements, err := docparser.ParseDocument("./train_data_Sila/data.docx")
	if err != nil {
		log.Fatalf("Ошибка парсинга документа: %v", err)
	}

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	// Обслуживание статических файлов (изображений)
	router.Static("/images", "./images")

	// Инициализируем обработчики
	handlers.InitHandlers(router, elements, iamToken)

	router.Run(":8080")
}
