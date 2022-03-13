package auth

import (
	"bytes"
	"context"
	"encoding/json"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/gulfcoastdevops/snow/config"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type Profile struct {
	Login     string `json:"login"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Locale    string `json:"locale"`
	TimeZone  string `json:"timeZone"`
}
type User struct {
	Id              string    `json:"id"`
	PasswordChanged time.Time `json:"passwordChanged"`
	Profile         `json:"profile"`
}
type Embedded struct {
	User `json:"user"`
}
type Hints struct {
	Allow []string `json:"allow"`
}
type Cancel struct {
	Href  string `json:"href"`
	Hints `json:"hints"`
}
type Links struct {
	Cancel `json:"cancel"`
}
type Response struct {
	ExpiresAt    time.Time `json:"expiresAt"`
	Status       string    `json:"status"`
	SessionToken string    `json:"sessionToken"`
	Embedded     `json:"_embedded"`
	Links        `json:"_links"`
}
type Idp struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}
type LinksSession struct {
	Self        `json:"self"`
	Refresh     `json:"refresh"`
	UserSession `json:"user"`
}
type Self struct {
	Href  string `json:"href"`
	Hints `json:"hints"`
}
type Refresh struct {
	Href  string `json:"href"`
	Hints `json:"hints"`
}
type UserSession struct {
	Name  string `json:"name"`
	Href  string `json:"href"`
	Hints `json:"hints"`
}
type SessionRes struct {
	Id                       string    `json:"id"`
	UserId                   string    `json:"userId"`
	Login                    string    `json:"login"`
	CreatedAt                time.Time `json:"createdAt"`
	ExpiresAt                time.Time `json:"expiresAt"`
	Status                   string    `json:"status"`
	LastPasswordVerification time.Time `json:"lastPasswordVerification"`
	LastFactorVerification   time.Time `json:"lastFactorVerification"`
	Amr                      []string  `json:"amr"`
	Idp                      `json:"idp"`
	MfaActive                bool `json:"mfaActive"`
	LinksSession             `json:"_links"`
}

func getOktaDomain() string {
	configPath := config.GetConfigPath(os.Getenv("config"))
	cfg, err := config.GetConfig(configPath)
	if err != nil {
		log.Fatalf("Loading config: %v", err)
	}

	return cfg.Okta.Host
}

func getOktaApiToken() string {
	configPath := config.GetConfigPath(os.Getenv("config"))
	cfg, err := config.GetConfig(configPath)
	if err != nil {
		log.Fatalf("Loading config: %v", err)
	}
	return cfg.Okta.ApiToken
}

func ValidatesUser(username, password string) (*Response, error) {

	reqbody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	payload := bytes.NewBuffer(reqbody)

	res, err := http.Post(getOktaDomain()+"authn", "application/json", payload)
	if err != nil || res.Status == "401" {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()

	var user Response
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, err
	}
	if user.Status != "SUCCESS" {
		return nil, err
	}
	return &user, nil
}

func GetSessionToken(sessionToken string) (*SessionRes, error) {

	reqbody, _ := json.Marshal(map[string]string{
		"sessionToken": sessionToken,
	})
	payload := bytes.NewBuffer(reqbody)

	res, err := http.Post(getOktaDomain()+"sessions", "application/json", payload)
	if err != nil || res.Status == "401" {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()

	var user SessionRes
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func CheckSessionId(ctx context.Context) (*SessionRes, error) {

	sessionId, err := grpc_auth.AuthFromMD(ctx, "Token")

	req, err := http.NewRequest("GET", getOktaDomain()+"sessions/"+sessionId, nil)
	if err != nil || len(sessionId) == 0 {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", getOktaApiToken())

	res, err := http.DefaultClient.Do(req)
	if err != nil || res.Status == "401" {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	var session SessionRes
	err = json.Unmarshal(body, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
