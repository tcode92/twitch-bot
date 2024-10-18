package bot

import "fmt"

// ANSI
const reset = "\033[0m"
const whiteBg = "\033[47m"
const whiteTxt = "\033[37m"
const blackBg = "\033[40m"
const blackTxt = "\033[30m"

func (b *Bot) PrintPretty(msg *ChatMsg) {
	fmt.Printf("%s%s %s %s%s %s %s%s %s\n%s\n\n", blackBg, whiteTxt, msg.Channel, whiteBg, blackTxt, msg.User, reset, whiteTxt, reset, msg.Message)
}
