// commands
package main

import (
	"fmt"
	"strings"
	"regexp"
)

func (bot *Bot) CmdInterpreter(channel string, username string, usermessage string) {
	//regexp we are using to find codes
	reg := "([A-Za-z0-9]{4})-0000-00([A-Za-z0-9]{2})-([A-Za-z0-9]{4})"
	re := regexp.MustCompile(reg)
	matched, _ := regexp.MatchString(reg, usermessage)

	if matched {
		levelid := re.FindString(usermessage)
		//Remove the levelid from the message
		msg := strings.Replace(usermessage, levelid, "", 1)
		fmt.Printf("Triggered @ %s: %s and %s\n", channel,username,levelid)
		bot.writeLevelDB(channel,username,msg,levelid)
	}

	if strings.HasPrefix(usermessage, "!level") {
		if bot.isStreamer(username,channel) {
			message := strings.Replace(usermessage, "!level", "", 1)
			bot.Message(channel,bot.getLevel(true,channel,message))
		} else {
			bot.Message(channel,bot.getLevel(false,channel,""))
		}
	} else if strings.HasPrefix(usermessage, "!reroll") {
		if bot.isStreamer(username,channel) {
			bot.Message(channel,bot.doReroll(channel))
		}
	} else if strings.HasPrefix(usermessage, "!skip") {
		if bot.isStreamer(username,channel) {
			message := strings.Replace(usermessage, "!skip", "", 1)
                        bot.Message(channel,bot.doSkip(channel,message))
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

func (bot *Bot) isStreamer(username string, channel string) bool {
        temp := strings.Replace(channel, "#", "", 1)
        if temp == username {
                return true
        }
        return false
}

