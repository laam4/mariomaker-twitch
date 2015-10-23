# Super Mario Maker twitch bot
Twitch bot written in golang which collects Super Mario Maker level codes and adds them to MySQL database.

IRC code Based on https://github.com/Vaultpls/Twitch-IRC-Bot

## Features
- Reads messages from twitch chat and saves levelcodes to MySQL database, one code per message, no duplicates
- Everytime streamer uses `!level` command a new random level is selected from database
- Viewers can get last selected level with the same `!level` command
- `!reroll` command for rerolling a level and putting the current chosen level back to the random pool
- `!skip` command for skipping a level, eg. submitted level not beaten, wrong code etc.
- `!level` and `!skip` commands supports optional [comment] parameter, which affects the currently chosen level.
- Multichannel support and Website support
- Checks if the level submitter is watching the stream!

## Install
- Go to your go project folder
- Get dependencies `go get github.com/go-sql-driver/mysql` `go get github.com/fatih/color` `go get github.com/vharitonsky/iniflags`
- Get twitch bot `go get github.com/laam4/mariomaker-twitch`
- Create database and user to MySQL
- Edit `default.ini`
- Type `go install github.com/laam4/mariomaker-twitch`
- Run bot from your GOPATH/bin folder with -config parameter

## TODO:
- Code cleanup and better parsing
- Error handling
- Website http://mario.laama.dy.fi/
- More commands
- Whisper support
