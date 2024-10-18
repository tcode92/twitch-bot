package bot

import "github.com/tcode92/twitch-bot/ws"

type Bot struct {
	OnMessage     func(message ChatMsg)
	OnChannelJoin func(channel JoinChan)
	env           *Env
	client        *ws.Client
}

func New(env *Env) *Bot {
	return &Bot{
		OnMessage:     func(message ChatMsg) {},
		OnChannelJoin: func(channel JoinChan) {},
		env:           env,
		client:        nil,
	}
}
