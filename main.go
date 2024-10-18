package main

import (
	"bot/cmd/bot"
	"bot/cmd/twitch"
	"os"
	"strings"
)

func main() {
	// parse program arguments and load env from env files
	env := bot.GetEnv()
	// twitch api to authenticate and validate tokens
	twitch := twitch.New(env)
	// if tokens doesn't exists for any reason the user need to authenticate via browser
	if env.AccessToken == "" || env.RefreshToken == "" {
		err := twitch.AuthorizationCodeGrantFlow()
		if err != nil {
			println(err.Error())
			os.Exit(1)
		}
	} else {
		// validate the tokens and refresh access token if not valid
		if err := twitch.ValidateToken(); err != nil {
			println(err.Error())
			if err.Error() == "invalid access token" {
				err := twitch.RefreshAccessToken()
				if err != nil {
					println(err)
					os.Exit(1)
				}
			}
		}
	}

	b := bot.New(env)
	b.OnMessage = func(m bot.ChatMsg) {
		b.PrintPretty(&m)
		if m.User != env.UserName {
			if k := findKappas(m.Message); len(k) > 0 {
				b.SendMessage(m.Channel, strings.Join(k, " "))
			}
		}
	}
	b.OnChannelJoin = func(channel bot.JoinChan) {
		println("Joined channel: ", channel)
	}
	// connect to irc
	exit := b.Connect()

	// wait irc signal to exit
	<-exit
}

var kappas = []string{"Kappa", "KappaPride", "KappaClaus", "KappaRoss", "KappaWealth", "Keepo"}

func isKappa(str string) bool {
	for _, v := range kappas {
		if v == str {
			return true
		}
	}
	return false
}

func findKappas(m string) []string {
	k := []string{}
	for _, w := range strings.Split(m, " ") {
		if isKappa(w) {
			k = append(k, w)
		}
	}
	return k
}
