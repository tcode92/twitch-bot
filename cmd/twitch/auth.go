package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func (t *TwitchApi) AuthorizationCodeGrantFlow() error {

	authReq := url.Values{}
	authReq.Set("client_id", t.env.ClientId)
	authReq.Set("redirect_uri", t.env.RedirectUrl)
	authReq.Set("response_type", "code")
	authReq.Set("scope", "user:read:chat user:write:chat user:edit user:manage:chat_color user:read:emotes user:write:chat chat:edit chat:read")

	println("Please authorize the application through this link\n")
	println(fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?%s\n\n", authReq.Encode()))

	codeCh := make(chan string)
	url, err := url.Parse(t.env.RedirectUrl)
	if err != nil {
		println(err.Error())
		os.Exit(0)
	}
	go authHttpServer(codeCh, ":"+strings.Split(url.Host, ":")[1])

	code := <-codeCh

	return t.ExchangeCodeWithToken(code)
}

func (t *TwitchApi) ExchangeCodeWithToken(code string) error {
	var token TokenResponse
	body := url.Values{}
	body.Set("client_id", t.env.ClientId)
	body.Set("client_secret", t.env.ClientSecret)
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("redirect_uri", t.env.RedirectUrl)
	resp, err := http.Post("https://id.twitch.tv/oauth2/token", "application/x-www-form-urlencoded", strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		e := tokenError{}
		json.Unmarshal(respBody, &e)
		return fmt.Errorf("twitch auth error: %s", e.Message)
	}
	json.Unmarshal(respBody, &token)
	t.env.AccessToken = token.AccessToken
	t.env.RefreshToken = token.RefreshToken
	t.env.UpdateTokens()
	return nil
}

func (t *TwitchApi) RefreshAccessToken() error {

	body := url.Values{}
	body.Set("client_id", t.env.ClientId)
	body.Set("client_secret", t.env.ClientSecret)
	body.Set("grant_type", "refresh_token")
	body.Set("refresh_token", t.env.RefreshToken)
	resp, err := http.Post("https://id.twitch.tv/oauth2/token", "application/x-www-form-urlencoded", strings.NewReader(body.Encode()))
	var token RefreshTokenResponse
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		e := tokenError{}
		json.Unmarshal(respBody, &e)
		return fmt.Errorf("twitch auth error: %s", e.Message)
	}

	json.Unmarshal(respBody, &token)
	t.env.AccessToken = token.AccessToken
	t.env.RefreshToken = token.RefreshToken
	t.env.UpdateTokens()
	return nil

}

func (t *TwitchApi) ValidateToken() error {

	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("OAuth %s", t.env.AccessToken))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		return nil
	} else {
		m := tokenError{}
		err = json.Unmarshal(b, &m)
		if err != nil {
			return err
		}
		return errors.New(m.Message)
	}
}

func authHttpServer(codeCh chan string, addr string) {
	ctx := context.Background()
	s := &http.Server{
		Addr: addr,
	}
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		w.WriteHeader(200)
		if code == "" {
			println("Invalid code response.")
			os.Exit(1)
		}
		codeCh <- code
		s.Shutdown(ctx)
	})
	s.ListenAndServe()
}
