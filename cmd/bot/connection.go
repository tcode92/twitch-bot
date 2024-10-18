package bot

import (
	"bot/cmd/ws"
	"fmt"
	"regexp"
	"strings"
	"time"
)

func (b *Bot) Connect() chan interface{} {
	exitChan := make(chan interface{})
	go func() {
		loggedIn := make(chan interface{})
		client, err := ws.NewClient("wss://irc-ws.chat.twitch.tv:443/")
		b.client = client
		client.OnDisconnect = func() {
			println("Disconnected.")
		}
		client.OnTextMessage = func(ircMsg string) {
			m := b.parseIrcMsg(ircMsg)
			switch v := m.(type) {
			case LoginError:
				println(v)
				exitChan <- nil
			case ChatMsg:
				go b.OnMessage(v)
			case JoinChan:
				go b.OnChannelJoin(v)
			case Ping:
				go client.SendText("PONG :tmi.twitch.tv")
			case Login:
				println("Login sucessful")
				loggedIn <- nil

			}
		}
		client.OnPong = func() {
			println("got pong")
		}
		if err != nil {
			println(err.Error())
			exitChan <- nil
		}
		err = client.Connect()
		if err != nil {
			println(err.Error())
			exitChan <- nil
		}

		println("Connected")
		client.SendText(fmt.Sprintf("PASS oauth:%s", env.AccessToken))
		client.SendText(fmt.Sprintf("NICK %s", env.UserName))

		select {
		case <-loggedIn:
			// join all channels in env.
			if len(env.Channels) == 0 {
				println("No channels to join.")
				exitChan <- nil
			}
			for _, c := range env.Channels {
				client.SendText(fmt.Sprintf("JOIN #%s", c))
			}
		case <-time.After(time.Second * 10):
			exitChan <- nil
		}
	}()
	return exitChan
}

func (b *Bot) SendMessage(channel string, message string) {
	if b.client != nil {
		b.client.SendText(fmt.Sprintf("PRIVMSG #%s :%s", channel, message))
	}
}

type JoinChan string
type Login struct{}
type LoginError string
type Ping struct{}
type ChatMsg struct {
	Channel string
	User    string
	Message string
	Kappa   []string
}

var joinChanRE = regexp.MustCompile(`^:.*#(\w+)$`)
var chatMsg = regexp.MustCompile(`^:(.*)!.*PRIVMSG #(\w+) :(.*)`)

func (b *Bot) parseIrcMsg(msg string) interface{} {
	parts := strings.Split(msg, "\r\n")
	if len(parts) == 0 {
		return nil
	}
	if parts[0] == ":tmi.twitch.tv NOTICE * :Login authentication failed" {
		return LoginError("Login authentication failed")
	}
	if parts[0] == "PING :tmi.twitch.tv" {
		return Ping{}
	}

	if strings.HasPrefix(parts[0], ":tmi.twitch.tv 001") {
		return Login{}
	}
	if strings.HasPrefix(parts[0], fmt.Sprintf(":%s!%s@%s.tmi.twitch.tv JOIN", b.env.UserName, b.env.UserName, b.env.UserName)) {
		matches := joinChanRE.FindStringSubmatch(parts[0])
		if len(matches) == 2 {
			// this panics i know but we gonna trust
			return JoinChan(matches[1])
		}
		return JoinChan("Unknown")
	}
	if matches := chatMsg.FindStringSubmatch(parts[0]); len(matches) >= 4 {
		m := ChatMsg{}
		m.User = matches[1]
		m.Channel = matches[2]
		m.Message = matches[3]
		return m
	}
	return nil
}
