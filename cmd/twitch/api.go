package twitch

import "github.com/tcode92/twitch-bot/cmd/bot"

type TwitchApi struct {
	env *bot.Env
}

func New(env *bot.Env) TwitchApi {
	return TwitchApi{
		env: env,
	}
}
