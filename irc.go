package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/textproto"
	"os"
	"strings"
	"time"
	"log"
	"github.com/fatih/color"
)

var red = color.New(color.FgRed).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()
var magenta = color.New(color.FgMagenta).SprintFunc()
var cyan = color.New(color.FgCyan).SprintFunc()
var white = color.New(color.FgWhite).SprintFunc()

type Bot struct {
	server		string
	port		string
	nick		string
	channel		map[string]int
	conn		net.Conn
//	mods		map[string]bool
	lastmsg		int64
	maxMsgTime	int64
	//Save last random level
	levelId		map[int]int
        userName	map[int]string
        level		map[int]string
	//database
        user            string
        pass            string
        host            string
        database        string
}

func NewBot() *Bot {
	return &Bot{
		server:         "irc.twitch.tv",
		port:           "6667",
		nick:           "V4delma",
		channel:	make(map[string]int),
		conn:           nil,
//		mods:           make(map[string]bool),
		lastmsg:	0,
		maxMsgTime:	5,
		levelId:	make(map[int]int),
		userName:	make(map[int]string),
		level:		make(map[int]string),
		user:		"mario",
		pass:		"salakala",
		host:		"unix(/var/run/mysqld/mysqld.sock)",
		database:	"mariomaker",
	}
}

func (bot *Bot) Connect() {
	var err error
	color.Yellow("Connecting...\n")
	bot.conn, err = net.Dial("tcp", bot.server+":"+bot.port)
	if err != nil {
		color.Red("Unable to connect to Twitch IRC server! Reconnecting in 10 seconds...\n")
		time.Sleep(10 * time.Second)
		bot.Connect()
	}
	color.Green("Connected to IRC server %s\n", bot.server)
}

func (bot *Bot) Message(channel string, message string) {
	if message == "" {
		return
	}
	if bot.lastmsg+bot.maxMsgTime <= time.Now().Unix() {
		fmt.Printf("[%s] %s <%s> %s\n", time.Now().Format("15:04"), blue(channel), bot.nick, white(message))
		fmt.Fprintf(bot.conn, "PRIVMSG "+channel+" :"+message+"\r\n")
		bot.lastmsg = time.Now().Unix()
	} else {
		color.Yellow("Attempted to spam message")
		//Sleep 3s and send last message again
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
		if strings.HasPrefix(text, "#") {
			fmt.Fprintf(bot.conn, "PRIVMSG "+text+"\r\n")
		}
	}
}

func getColor(c string, name string) string {
	switch(c) {
		case "#0000FF", "#5F9EA0":
			return blue(name)
		case "#FF0000", "#B22222":
			return red(name)
		case "#8A2BE2", "#FF69B4":
			return magenta(name)
		case "#008000", "#00FF7F", "#2E8B57":
			return green(name)
		case "#DAA520", "#FF4500", "#D2691E":
			return yellow(name)
		case "#1E90FF":
			return cyan(name)
		default:
			return white(name)
	}
}

func main() {
	ircbot := NewBot()
	go ircbot.ConsoleInput()
	ircbot.Connect()

	ircbot.channel["#retku"] = 1
	ircbot.channel["#firnwath"] = 2
	ircbot.channel["#herramustikka"] = 3

	info := color.New(color.FgWhite, color.BgGreen).SprintFunc()

	pass1, err := ioutil.ReadFile("twitch_pass.txt")
	pass := strings.Replace(string(pass1), "\n", "", 0)
	if err != nil {
		fmt.Println("Error reading from twitch_pass.txt.  Maybe it isn't created?")
		os.Exit(1)
	}
	//((Connecting..
	fmt.Fprintf(ircbot.conn, "USER %s 8 * :%s\r\n", ircbot.nick, ircbot.nick)
	fmt.Fprintf(ircbot.conn, "PASS %s\r\n", pass)
	fmt.Fprintf(ircbot.conn, "NICK %s\r\n", ircbot.nick)
	fmt.Fprintf(ircbot.conn, "CAP REQ :twitch.tv/membership twitch.tv/tags twitch.tv/commands\r\n")
	
	fmt.Printf("Channels: ")
	//Looping through all the channels
	for k, i := range ircbot.channel {
		fmt.Fprintf(ircbot.conn, "JOIN %s\r\n", k)
		fmt.Printf("#%d: %s, ", i, blue(k))
	}
	fmt.Printf("\nInserted information to server...\n")
	//Initialize DB = Create tables & add streamers
	ircbot.InitDB()
	defer ircbot.conn.Close()

	reader2 := bufio.NewReader(ircbot.conn)
	tp := textproto.NewReader(reader2)
	go ircbot.ConsoleInput()
	for {
		line, err := tp.ReadLine()
		if err != nil {
			break // break loop on errors
		}
		for k, i := range ircbot.channel {
			if strings.Contains(line, k) {
			f, err := os.OpenFile("logs/"+k, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
				if err != nil {
    					log.Fatalf("error opening %s%d file: %v", k, i, err)
				}
			//defer f.Close()
			log.SetOutput(f)
			log.Println(line)
			f.Close()
			}
		}
		if strings.Contains(line, "PING") {
			//Twitch always gives PING tmi.twitch.tv, may change this to read the PING line
			fmt.Fprintf(ircbot.conn, "PONG tmi.twitch.tv\r\n")
		} else if strings.Contains(line, ".tmi.twitch.tv PRIVMSG") {
                        userdata := strings.Split(line, ".tmi.twitch.tv PRIVMSG ")
			chanmsg := strings.SplitN(userdata[1], " :", 2)
                        tags := strings.Split(userdata[0], ";")
			if strings.Contains(tags[0], "twitchnotify") {
				username := strings.Split(tags[0], "@")
				fmt.Printf("[%s] %s %s %s\n", time.Now().Format("15:04"), blue(chanmsg[0]),  info(username[1]), white(chanmsg[1]))
			} else {
				dispname := strings.Replace(tags[1], "display-name=", "", 1)
				var username string
				if dispname == "" {
					name := strings.Split(userdata[0], "@")
					username = name[2]
				} else {
					username = dispname
				}
                        	color := strings.Replace(tags[0], "@color=", "", 1)
				fmt.Printf("[%s] %s <%s> %s\n", time.Now().Format("15:04"), blue(chanmsg[0]), getColor(color,username), white(chanmsg[1]))
				go ircbot.CmdInterpreter(chanmsg[0], username, chanmsg[1])
			}
		}
	}
}
