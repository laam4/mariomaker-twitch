// commands
package main

import (
	"fmt"
	"strings"
	"regexp"
	"net/http"
	"io/ioutil"
	"net/url"
	"log"
	"time"
)

var lastchat int64 = 0

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
	} else if strings.Contains(usermessage, bot.nick) {
		msg := strings.Replace(usermessage, bot.nick, "", 1)
		if msg != "" && lastchat+10 <= time.Now().Unix() {
			bot.Message(channel,bot.askOracle(username,msg))
			lastchat = time.Now().Unix()
		}
	} else if strings.HasPrefix(usermessage, "!level") {
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
/*func (bot *Bot) isMod(username string, channel string) bool {
	temp := strings.Replace(channel, "#", "", 1)
	if bot.mods[username] == true || temp == username {
		return true
	}
	return false
}
*/

func (bot *Bot) isStreamer(username string, channel string) bool {
        temp := strings.Replace(channel, "#", "", 1)
        if temp == strings.ToLower(username) {
                return true
        }
        return false
}

func (bot *Bot) askOracle(username string, message string) string {
	pohja := "http://www.lintukoto.net/viihde/oraakkeli/index.php?kysymys="
	url := pohja + url.QueryEscape(message) + "&html"
	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("Cannot get URL response: %s\n", err.Error())
	} else {
        	defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
		log.Fatalf("Cannot read URL response: %s\n", err.Error())
		}
		answer := strings.Replace(toUtf8(contents), "\n", "", 1)
		result := fmt.Sprintf("%s: %s", username, answer)
		return result
	}
	return "Kappa"
}

func toUtf8(iso8859_1_buf []byte) string {
    buf := make([]rune, len(iso8859_1_buf))
    for i, b := range iso8859_1_buf {
        buf[i] = rune(b)
    }
    return string(buf)
}
