package docparser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"assistant-bot/pkg/models"
)

func ParseDocument(filePath string) ([]models.DocumentElement, error) {
	var elements []models.DocumentElement

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
			elements = append(elements, models.DocumentElement{
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
				elements = append(elements, models.DocumentElement{
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
