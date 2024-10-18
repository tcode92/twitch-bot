package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func (t *TwitchApi) GetUserInfo(user string) (UserInfo, error) {
	var u = TwitchResponse[UserInfo]{}
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", user), nil)
	if err != nil {
		return UserInfo{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.env.AccessToken))
	req.Header.Set("Client-Id", t.env.ClientId)

	resp, err := client.Do(req)
	if err != nil {
		return UserInfo{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserInfo{}, err
	}
	if resp.StatusCode == 200 {
		err := json.Unmarshal(b, &u)
		if err != nil {
			return UserInfo{}, errors.New("error parsing json response")
		}
		return u.Data[0], nil

	} else {
		return UserInfo{}, errors.New("invalid response")
	}
}
