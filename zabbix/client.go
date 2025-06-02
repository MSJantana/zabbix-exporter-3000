package zabbix

import (
	"crypto/tls"
	"encoding/json"
	"github.com/cavaliercoder/go-zabbix"
	cnf "github.com/MSJantana/zabbix-exporter-3000/config"
	"log"
	"net/http"
	"strings"
)

var Session, err = Connect()
var Query *zabbix.Request

func Connect() (*zabbix.Session, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cnf.SslSkip,
			},
		},
	}

	cache := zabbix.NewSessionFileCache().SetFilePath("./zabbix_session")

	session := zabbix.CreateClient(cnf.Server).
		WithCache(cache).
		WithHTTPClient(client)

	// Corrige payload para Zabbix 7+
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "user.login",
		"params": map[string]interface{}{
			"username": cnf.User,
			"password": cnf.Password,
		},
		"id":   1,
		"auth": nil,
	}

	resp, err := session.CallRaw(payload)
	if err != nil {
		log.Fatalf("Erro ao autenticar no Zabbix: %v", err)
	}

	authToken, ok := resp.Result.(string)
	if !ok {
		log.Fatalf("Erro ao extrair o token de autenticação: %v", resp)
	}

	session.SetAuthToken(authToken)

	version, err := session.GetVersion()
	if err != nil {
		log.Fatalf("Erro ao obter a versão do Zabbix: %v", err)
	}

	strRequestWithAuth := strings.Replace(cnf.Query, "%auth-token%", authToken, -1)

	err = json.Unmarshal([]byte(strRequestWithAuth), &Query)
	if err != nil {
		log.Printf("Erro ao converter JSON da query: %v", err)
	}

	log.Printf("Conectado ao Zabbix API v%s", version)
	return session, nil
}
