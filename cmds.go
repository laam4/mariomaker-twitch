// commands
package main

import (
	"fmt"
	"strings"
	"regexp"
)

func (bot *Bot) CmdInterpreter(channel string, username string, usermessage string) {
	//message := strings.ToLower(usermessage)

	reg := "([A-Za-z0-9]{4})-0000-00([A-Za-z0-9]{2})-([A-Za-z0-9]{4})"
	//tempstr := strings.Split(message, " ")
	//reg := "([A-Za-z0-9]{4})-([A-Za-z0-9]{4})-([A-Za-z0-9]{4})-([A-Za-z0-9]{4})"
	re := regexp.MustCompile(reg)
	matched, _ := regexp.MatchString(reg, usermessage)

	if matched {
		levelid := re.FindString(usermessage)
		messige := strings.Replace(usermessage, levelid, "", 1)
		fmt.Printf("Triggered @ %s: %s and %s\n", channel,username,levelid)
		bot.writeLevelDB(channel,username,messige,levelid)
	}

	if strings.HasPrefix(usermessage, "!level") {
		if bot.isMod(username,channel) {
			bot.Message(channel,bot.getLevel(true,channel))
		} else {
			bot.Message(channel,bot.getLevel(false,channel))
		}
	} else if strings.HasPrefix(usermessage, "!stats") {
		bot.Message(channel,bot.getStats(channel))
	}
}
//Mod stuff
func (bot *Bot) isMod(username string, channel string) bool {
	temp := strings.Replace(channel, "#", "", 1)
	if bot.mods[username] == true || temp == username {
		return true
	}
	return false
}
