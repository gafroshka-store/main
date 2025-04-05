package wrappers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

const (
	authURL = "http://localhost:8081" // позже поменяем localhost на имя контейнера
)

type AuthWrapper struct {
	BaseURL string
}

type AuthWrapperRepo interface {
	Auth(username, password string) (string, error)
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	Error string `json:"error,omitempty"`
}

func NewAuthWrapper() *AuthWrapper {
	return &AuthWrapper{
		BaseURL: authURL,
	}
}

// Auth обертка на запрос в сервис авторизации
func (aw *AuthWrapper) Auth(username, password string) (string, error) {
	url := aw.BaseURL + "/auth/login"
	requestBody, err := json.Marshal(AuthRequest{Username: username, Password: password})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(authResp.Error)
	}

	return authResp.Token, nil
}
