package storage

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"assistant-bot/pkg/models"
)

func SaveQA(filePath string, qa models.QA) error {
	var qas []models.QA

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

func ClearJSONFile(filePath string) error {
	emptyArray := []models.QA{}
	data, err := json.MarshalIndent(emptyArray, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func LoadQA(filePath string) ([]models.QA, error) {
	var qas []models.QA

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &qas); err != nil {
		return nil, err
	}

	return qas, nil
}
