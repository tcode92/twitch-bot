package bot

import "bot/cmd/ws"

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
