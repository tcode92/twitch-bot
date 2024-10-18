package twitch

import "bot/cmd/bot"

type TwitchApi struct {
	env *bot.Env
}

func New(env *bot.Env) TwitchApi {
	return TwitchApi{
		env: env,
	}
}
