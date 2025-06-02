package zabbix

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"github.com/rzrbld/zabbix-exporter-3000/config"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var AuthToken string
var Query map[string]interface{}

func Connect() (*http.Client, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.SslSkip,
			},
		},
	}

	// Login com username (compatível com Zabbix 6.0/7.0)
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "user.login",
		"params": map[string]string{
			"username": config.User,   // ← CORRETO!
			"password": config.Password,
		},
		"id":   1,
		"auth": nil,
	}

	body, _ := json.Marshal(payload)

	resp, err := client.Post(config.Server, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Erro ao fazer login no Zabbix: %v", err)
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		log.Fatalf("Erro ao parsear resposta JSON: %v", err)
	}

	token, ok := result["result"].(string)
	if !ok {
		log.Fatalf("Falha ao obter token de autenticação: %s", data)
	}

	AuthToken = token
	log.Printf("Token de autenticação recebido com sucesso.")

	// Substitui %auth-token% na query configurada
	strRequestWithAuth := strings.Replace(config.Query, "%auth-token%", token, -1)

	// Transforma query para o formato JSON
	if err := json.Unmarshal([]byte(strRequestWithAuth), &Query); err != nil {
		log.Print("Erro ao converter Query para JSON: ", err)
	}

	return client, nil
}
