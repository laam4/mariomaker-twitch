# mariomaker-twitch
Twitch bot written in golang which collects Super Mario Maker level codes and adds them to MySQL database

Based on https://github.com/Vaultpls/Twitch-IRC-Bot

## Install
- Go to your go project folder
- Type `go get github.com/laam4/mariomaker-twitch`
- Edit irc.go with channel, bot, and mysql details
- Create `twitch_pass.txt` with oauth login for bot
- Type `go install github.com/laam4/mariomaker-twitch`
- Run bot from your GOPATH/bin folder

## TODO:
- Code cleanup [Almost done]
- Error handling [Done?]
- Bacon [Mmmm...]
- Website http://mario.laama.dy.fi/
- More commands
- Better logic for SQL queries [Done]
- Configuration file?
