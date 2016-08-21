// commands
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var lastchat int64

//CmdInterpreter Receives PRIVMSGs from all the channels and interprets all the commands
func CmdInterpreter(channel string, username string, usermessage string) {
	//regexp we are using to find codes
	reg := "([[:xdigit:]]{4})-0000-([[:xdigit:]]{4})-([[:xdigit:]]{4})"
	re := regexp.MustCompile(reg)
	matched, _ := regexp.MatchString(reg, usermessage)

	if matched {
		levelid := re.FindString(usermessage)
		//Remove the levelid from the message
		msg := strings.Replace(usermessage, levelid, "", 1)
		fmt.Printf("Triggered @ %s: %s and %s\n", channel, username, levelid)
		writeLevelDB(channel, username, msg, levelid)
	} else if username == "twitchnotify" && !strings.Contains(usermessage, "subscribed to") {
		split := strings.Split(usermessage, " ")
		var name string
		var months string
		if strings.HasSuffix(usermessage, "row!") {
			name = split[0]
			months = split[3]
		} else if strings.HasSuffix(usermessage, "subscribed!") {
			name = split[0]
			months = "1"

		}
		if name != "" {
			//fmt.Printf("%q %q\n", name, months)
			writeSubs(channel, name, months)
		}
	} else if channel != "#retku" && strings.Contains(usermessage, nick) || channel != "#retku" && strings.Contains(usermessage, strings.ToLower(nick)) {
		msg := strings.Replace(usermessage, nick, "", 1)
		if msg != "" && lastchat+10 <= time.Now().Unix() {
			//if msg != "" {
			Message(channel, askOracle(username, msg))
			lastchat = time.Now().Unix()
		}
	} else if strings.HasPrefix(usermessage, "!level") {
		if isStreamer(username, channel) {
			message := strings.Replace(usermessage, "!level", "", 1)
			Message(channel, getLevel(true, channel, message))
		} else {
			Message(channel, getLevel(false, channel, ""))
		}
	} else if strings.HasPrefix(usermessage, "!reroll") {
		if isStreamer(username, channel) {
			Message(channel, doReroll(channel))
		}
	} else if strings.HasPrefix(usermessage, "!skip") {
		if isStreamer(username, channel) {
			message := strings.Replace(usermessage, "!skip", "", 1)
			Message(channel, doSkip(channel, message))
		}
	} else if strings.HasPrefix(usermessage, "!mariostats") {
		Message(channel, getStats(channel))
	}
}

func isStreamer(username string, channel string) bool {
	temp := strings.Replace(channel, "#", "", 1)
	if temp == strings.ToLower(username) {
		return true
	}
	return false
}

func askOracle(username string, message string) (result string) {
	pohja := "http://www.lintukoto.net/viihde/oraakkeli/index.php?kysymys="
	url := pohja + url.QueryEscape(message) + "&html"
	response, err := http.Get(url)
	if err != nil {
		log.Printf("Cannot get URL response: %s\n", err.Error())
		result = fmt.Sprintf("%s: En osaa juuri nyt vastata, voit jättää viestin äänimerkin jälkeen", username)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("Cannot read URL response: %s\n", err.Error())
			result = fmt.Sprintf("%s: En osaa juuri nyt vastata, voit jättää viestin äänimerkin jälkeen", username)
		}
		//answer := strings.Replace(string(contents), "\n", "", 1)
		answer := strings.Replace(toUtf8(contents), "\n", "", 1)
		result = fmt.Sprintf("%s: %s", username, answer)
	}
	return
}

func toUtf8(isobuf []byte) string {
	buf := make([]rune, len(isobuf))
	for i, b := range isobuf {
		buf[i] = rune(b)
	}
	return string(buf)
}
