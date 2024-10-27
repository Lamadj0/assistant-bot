package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"assistant-bot/pkg/models"
)

func GenerateCompletion(userText, context, iamToken string) (string, error) {
	url := "https://llm.api.cloud.yandex.net/foundationModels/v1/completion"

	requestData := models.RequestData{
		ModelUri: "gpt://b1gjp5vama10h4due384/yandexgpt/latest",
		CompletionOptions: models.CompletionOptions{
			Stream:      false,
			Temperature: 0.6,
			MaxTokens:   2000,
		},
		Messages: []models.Message{
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

	var responseData models.ResponseData
	if err := json.Unmarshal(body, &responseData); err != nil {
		return "", err
	}

	if len(responseData.Result.Alternatives) > 0 {
		return responseData.Result.Alternatives[0].Message.Text, nil
	}

	return "Ответ не получен", nil
}
