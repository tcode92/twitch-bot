package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Env struct {
	userEnvPath  string
	ClientId     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
	UserName     string
	RedirectUrl  string
	Channels     []string
}

var env Env
var execAuth bool = false

func GetEnv() *Env {

	envPath, userEnvPath := parseArgs()

	var botEnv map[string]interface{}
	if envPath == "" {
		botEnv = readEnv(".env")
	} else {
		botEnv = readEnv(envPath)
	}

	// apply app env to env var
	v, ok := botEnv["CLIENT_ID"].(string)
	if ok {
		env.ClientId = v
	}
	v, ok = botEnv["CLIENT_SECRET"].(string)
	if ok {
		env.ClientSecret = v
	}
	v, ok = botEnv["REDIRECT_URL"].(string)
	if ok {
		env.RedirectUrl = v
	}
	if userEnvPath == "" {
		v, ok := botEnv["DEFAULT_USER"].(string)
		if ok {
			userEnvPath = v
		}
	}
	// parse env file
	if userEnvPath == "" {
		argError("Missing user env file.")
	}
	userEnv := readEnv(userEnvPath)
	v, ok = userEnv["ACCESS_TOKEN"].(string)
	if ok {
		env.AccessToken = v
	}
	v, ok = userEnv["REFRESH_TOKEN"].(string)
	if ok {
		env.RefreshToken = v
	}
	v, ok = userEnv["USER"].(string)
	if ok {
		env.UserName = v
	}
	c, ok := userEnv["CHAN"].([]string)
	if ok {
		env.Channels = c
	}
	env.userEnvPath = userEnvPath
	return &env
}

// parses program exec args to determin actions to be made
func parseArgs() (string, string) {
	var envPath string
	var userEnvPath string
	argLen := len(os.Args) - 1
	for i, arg := range os.Args {
		switch arg {
		case "--authorize":
			execAuth = true
		case "--env":
			if argLen < i+1 {
				argError("Missing env file")
			}
			envPath = os.Args[i+1]
		case "--user":
			if argLen < i+1 {
				argError("Missing user env file")
			}
			userEnvPath = os.Args[i+1]
		case "--init":
			if argLen < i+1 {
				argError("Please specify the env file path to generate")
			}
			newEnv :=
				`# Twitch application client id
CLIENT_ID=
# Twitch application client secret
CLIENT_SECRET=
# Twitch application Redirect url
REDIRECT_URL=
# Default user - Will be used if --user arg is not provided
DEFAULT_USER=
`
			f, err := filepath.Abs(os.Args[i+1])
			if err != nil {
				argError(err.Error())
			}
			err = os.WriteFile(f, []byte(newEnv), os.ModePerm)
			if err != nil {
				argError(err.Error())
			}
			fmt.Printf("New env file generated %s\n", f)
		case "--init-user":
			if argLen < i+1 {
				argError("Please specify the env file path for the user to generate")
			}
			newEnv :=
				`# User name
USER=
# User access token
ACCESS_TOKEN=
# User refresh token
REFRESH_TOKEN=
# Channels to join on connect
# Channels are specified as CHAN=channelname
# Multiple channels are possible by having multiple CHAN= key
CHAN=
`
			f, err := filepath.Abs(os.Args[i+1])
			if err != nil {
				argError(err.Error())
			}
			err = os.WriteFile(f, []byte(newEnv), os.ModePerm)
			if err != nil {
				argError(err.Error())
			}
			fmt.Printf("New user env file generated %s\n", f)
		}
	}
	if execAuth {
		println("NEED AUTH EXEC")
	}
	return envPath, userEnvPath
}

// read env file and returns a key value as key: string value: string or []string
func readEnv(p string) map[string]interface{} {
	b, err := os.ReadFile(p)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	c := strings.Split(string(b), "\n")
	e := map[string]interface{}{}
	for _, l := range c {
		trimmed := strings.TrimSpace(l)
		// ignore comments lines
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value := getLineKeyValue(l)
		if key == "CHAN" {
			chans, ok := e[key].([]string)
			if ok {
				e[key] = append(chans, value)
			} else {
				e[key] = []string{value}
			}
		} else if key != "" {
			e[key] = value
		}
	}
	return e
}

// parse env line and return key value
func getLineKeyValue(line string) (string, string) {
	i := strings.Index(line, "=")
	if i == -1 {
		return "", ""
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
}

// print string and exit the program
func argError(error string) {
	fmt.Printf("%s\n", error)
	os.Exit(1)
}

func (e *Env) UpdateTokens() {
	if e.userEnvPath == "" {
		println("User env file path is not defined.")
		return
	}
	f, err := filepath.Abs(e.userEnvPath)
	if err != nil {
		fmt.Printf("file %s not found\n", e.userEnvPath)
		return
	}
	println("Updating tokens")
	b, err := os.ReadFile(f)
	if err != nil {
		fmt.Printf("An error occurred while opening file %s\n", f)
		return
	}
	content := strings.Split(string(b), "\n")

	for i, l := range content {
		k, _ := getLineKeyValue(l)
		if k == "ACCESS_TOKEN" {
			content[i] = fmt.Sprintf("ACCESS_TOKEN=%s", e.AccessToken)
		}
		if k == "REFRESH_TOKEN" {
			content[i] = fmt.Sprintf("REFRESH_TOKEN=%s", e.RefreshToken)
		}
	}
	err = os.WriteFile(f, []byte(strings.Join(content, "\n")), os.ModePerm)
	if err != nil {
		fmt.Printf("An error occurred while writing file %s\n", f)
		return
	}
	fmt.Printf("Updated user env: %s\n", f)
}
