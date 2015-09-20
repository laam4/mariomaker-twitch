package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/textproto"
	"os"
	"strings"
	"time"
)

type Bot struct {
	server         string
	port           string
	nick           string
	channel        map[int]string
	conn           net.Conn
	levels         map[string]string
	mods           map[string]bool
	lastmsg        int64
	maxMsgTime     int64
        creator		string
        level		string
}

func NewBot() *Bot {
	return &Bot{
		server:         "irc.twitch.tv",
		port:           "6667",
		nick:           "V4delma",
		channel:	make(map[int]string),
		conn:           nil, //Don't change this
		levels:         make(map[string]string),
		mods:           make(map[string]bool),
		lastmsg:	0,
		maxMsgTime:	5,
	}
}

func (bot *Bot) Connect() {
	var err error
	fmt.Printf("Attempting to connect to server...\n")
	bot.conn, err = net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		fmt.Printf("Unable to connect to Twitch IRC server! Reconnecting in 10 seconds...\n")
		time.Sleep(10 * time.Second)
		bot.Connect()
	}
	fmt.Printf("Connected to IRC server %s\n", bot.server)
}

func (bot *Bot) Message(channel string, message string) {
	if message == "" {
		return
	}
	if bot.lastmsg+bot.maxMsgTime <= time.Now().Unix() {
		fmt.Printf("%s: %s\n", channel, message)
		fmt.Fprintf(bot.conn, "PRIVMSG "+channel+" :"+message+"\r\n")
		bot.lastmsg = time.Now().Unix()
	} else {
		fmt.Println("Attempted to spam message")
		time.Sleep(3 * time.Second)
		fmt.Fprintf(bot.conn, "PRIVMSG "+channel+" :"+message+"\r\n")
	}
}

func (bot *Bot) ConsoleInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		if text == "/quit" {
			bot.conn.Close()
			os.Exit(0)
		}
		if text == "/join" {
			kanava := strings.Replace(text, "/join ", "", 1)
			fmt.Fprintf(bot.conn, "JOIN %s\r\n", kanava)
		}
	}
}

func main() {
//	nick := flag.String("nickname", "V4delma", "The username of the bot.")
	flag.Parse()

	ircbot := NewBot()
	go ircbot.ConsoleInput()
	ircbot.Connect()

        ircbot.channel[0] = "#retku"
	ircbot.channel[1] = "#firnwath"

	pass1, err := ioutil.ReadFile("twitch_pass.txt")
	pass := strings.Replace(string(pass1), "\n", "", 0)
	if err != nil {
		fmt.Println("Error reading from twitch_pass.txt.  Maybe it isn't created?")
		os.Exit(1)
	}

	fmt.Fprintf(ircbot.conn, "USER %s 8 * :%s\r\n", ircbot.nick, ircbot.nick)
	fmt.Fprintf(ircbot.conn, "PASS %s\r\n", pass)
	fmt.Fprintf(ircbot.conn, "NICK %s\r\n", ircbot.nick)
	for i := range ircbot.channel {
		fmt.Fprintf(ircbot.conn, "JOIN %s\r\n", ircbot.channel[i])
		fmt.Printf("Joined: %s\n", ircbot.channel[i])
	}
	fmt.Printf("Inserted information to server...\n")
	ircbot.InitDB()
	//fmt.Printf("Channel: " + channel + "\n")
	defer ircbot.conn.Close()
	reader2 := bufio.NewReader(ircbot.conn)
	tp := textproto.NewReader(reader2)
	go ircbot.ConsoleInput()
	for {
		line, err := tp.ReadLine()
		if err != nil {
			break // break loop on errors
		}
		if strings.Contains(line, "PING") {
			//pongdata := strings.Split(line, "PING ")
			//fmt.Println(pongdata[1])
			fmt.Fprintf(ircbot.conn, "PONG tmi.twitch.tv\r\n")
		} else if strings.Contains(line, ".tmi.twitch.tv PRIVMSG") {
                        userdata := strings.Split(line, ".tmi.twitch.tv PRIVMSG ")
                        channel := strings.Split(userdata[1], " :")
                        username := strings.Split(userdata[0], "@")
                        usermessage := strings.Replace(userdata[1], channel[0]+" :", "", 1)
                        fmt.Printf(channel[0] +" "+ username[1] + ": " + usermessage + "\n")
			go ircbot.CmdInterpreter(channel[0], username[1], usermessage)

		} else if strings.Contains(line, ".tmi.twitch.tv JOIN ") {
			userjoindata := strings.Split(line, ".tmi.twitch.tv JOIN ")
			userjoined := strings.Split(userjoindata[0], "@")
			fmt.Printf(userjoined[1] + " has joined!\n")
		} else if strings.Contains(line, ".tmi.twitch.tv PART ") {
			userjoindata := strings.Split(line, ".tmi.twitch.tv PART ")
			userjoined := strings.Split(userjoindata[0], "@")
			fmt.Printf(userjoined[1] + " has left!\n")
		} else if strings.Contains(line, ":jtv MODE  +o ") {
			usermod := strings.Split(line, ":jtv MODE  +o ")
			ircbot.mods[usermod[1]] = true
			fmt.Printf(usermod[1] + " is a moderator!\n")
		} else if strings.Contains(line, ":jtv MODE  -o ") {
			usermod := strings.Split(line, ":jtv MODE  -o ")
			ircbot.mods[usermod[1]] = false
			fmt.Printf(usermod[1] + " isn't a moderator anymore!\n")
		}
	}

}
