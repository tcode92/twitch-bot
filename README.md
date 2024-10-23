# Twitch Bot

This Twitch bot is built using Go and requires two `.env` files to run: one for the application and one for the user.

## Building the Program

To build the program, use the following command:

```bash
go build -o twitchbot main.go
```

## Environment Files

The bot requires two `.env` files:

1. **Application `.env` file**: Contains configuration settings for the bot.
2. **User `.env` file**: Stores user-specific settings.

### Generate Environment Files

You can generate these required files with the following commands:

- For the application `.env` file:
  ```bash
  twitchbot --init <application/file/path>
  ```

- For the user `.env` file:
  ```bash
  twitchbot --init-user <user/file/path>
  ```

Both files are necessary for the bot to run properly.

## Running the Bot

The bot requires both the application and user `.env` files to be set. You can specify the user environment file in the application `.env` file or provide it directly during runtime.

### Default Run

By default, the bot will look for the `.env` file in the current directory. It will also check the `DEFAULT_USER` variable within the application `.env` file to load the user-specific file.

To run the bot:

```bash
twitchbot
```

### Custom Environment File Path

You can specify a custom environment file for the bot using the `--env` flag:

```bash
twitchbot --env <env/path>
```

In this case, the bot will look for the user `.env` file using the `DEFAULT_USER` variable in the specified environment file.

### Custom User File Path

To specify a custom user environment file, use the `--user` flag:

```bash
twitchbot --user <user/env/path>
```

### Full Custom Run

To run the bot with both custom environment and user files, use both `--env` and `--user` flags:

```bash
twitchbot --env <env/path> --user <user/env/path>
```

Make sure both the application and user `.env` files are configured correctly before running the bot.
