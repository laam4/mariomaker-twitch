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

var lastchat int64 = 0

func CmdInterpreter(channel string, username string, usermessage string) {
	//regexp we are using to find codes
	reg := "([[:xdigit:]]{4})-0000-00([[:xdigit:]]{2})-([[:xdigit:]]{4})"
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
	} else if strings.Contains(usermessage, nick) || strings.Contains(usermessage, strings.ToLower(nick)) {
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

func askOracle(username string, message string) string {
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
