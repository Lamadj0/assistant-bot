package utils

import (
	"assistant-bot/pkg/models"
	"regexp"
	"strings"
)

func IsInvalidQuestion(question string) bool {
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

func FindRelevantContext(question string, elements []models.DocumentElement) (string, []string, bool) {
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
