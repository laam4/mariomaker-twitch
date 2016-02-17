package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/vharitonsky/iniflags"
	"log"
	"math/rand"
	"net"
	"net/textproto"
	"os"
	"strings"
	"time"
)

var (
	red         = color.New(color.FgRed).SprintFunc()
	green       = color.New(color.FgGreen).SprintFunc()
	yellow      = color.New(color.FgYellow).SprintFunc()
	blue        = color.New(color.FgBlue).SprintFunc()
	magenta     = color.New(color.FgMagenta).SprintFunc()
	cyan        = color.New(color.FgCyan).SprintFunc()
	white       = color.New(color.FgWhite).SprintFunc()
	info        = color.New(color.FgWhite, color.BgGreen).SprintFunc()
	bg_magenta  = color.New(color.FgWhite, color.BgMagenta).SprintFunc()
	bg_yellow   = color.New(color.FgWhite, color.BgYellow).SprintFunc()
	server      string
	port        string
	nick        string
	channellist string
	database    string
	oauth       string
	debug       bool
	lastmsg     int64 = 0
	maxMsgTime  int64 = 5
	g_levelId   map[int]int
	g_userName  map[int]string
	g_level     map[int]string
	channels    map[string]int
	conn        net.Conn
)

func init() {
	flag.StringVar(&server, "server", "irc.twitch.tv", "IRC server address")
	flag.StringVar(&port, "port", "6667", "IRC server port")
	flag.StringVar(&nick, "nick", "Botname", "Bot's nickname")
	flag.StringVar(&channellist, "channellist", "#botname", "Comma separated list of channel to join")
	flag.StringVar(&database, "database", "username:password@protocol(address)/dbname?param=value", "MySQL Data Source Name")
	flag.StringVar(&oauth, "oauth", "oauth:token", "OAuth token for login, https://twitchapps.com/tmi/")
	flag.BoolVar(&debug, "debug", false, "If true prints all IRC messages from server as map")
	rand.Seed(time.Now().UnixNano())
}

func Connect() {
	var err error
	color.Yellow("Connecting...\n")
	conn, err = net.Dial("tcp", server+":"+port)
	if err != nil {
		color.Red("Unable to connect to Twitch IRC server! Reconnecting in 10 seconds...\n")
		time.Sleep(10 * time.Second)
		Connect()
	}
	color.Green("Connected to IRC server %s\n", server)
}

func Message(channel string, message string) {
	if message == "" {
		return
	}
	if lastmsg+maxMsgTime <= time.Now().Unix() {
		fmt.Printf("[%s] %s <%s> %s\n", time.Now().Format("15:04"), blue(channel), bg_magenta(nick), white(message))
		fmt.Fprintf(conn, "PRIVMSG "+channel+" :"+message+"\r\n")
		lastmsg = time.Now().Unix()
	} else {
		color.Yellow("Attempted to spam message")
		//Sleep 3s and send last message again
		time.Sleep(3 * time.Second)
		fmt.Fprintf(conn, "PRIVMSG "+channel+" :"+message+"\r\n")
	}
}

func ConsoleInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		if text == "/quit" {
			conn.Close()
			os.Exit(0)
		}
		if strings.HasPrefix(text, "#") {
			fmt.Fprintf(conn, "PRIVMSG "+text+"\r\n")
		}
	}
}

func fmtName(c string, name string, sub string, turbo string, utype string) string {
	var p string
	switch utype {
	case "mod", "globalmod":
		p = info("♣")
	case "staff":
		p = "S"
	case "admin":
		p = "A"
	}
	if sub == "1" {
		p = p + bg_yellow("☻")
	}
	if turbo == "1" {
		p = p + bg_magenta("T")
	}
	switch c {
	case "#0000FF", "#5F9EA0":
		name = blue(name)
	case "#FF0000", "#B22222", "#FF7F50", "#CC0000":
		name = red(name)
	case "#8A2BE2", "#FF69B4", "#FF6BB5":
		name = magenta(name)
	case "#008000", "#00FF7F", "#2E8B57", "#9ACD32":
		name = green(name)
	case "#DAA520", "#FF4500", "#D2691E", "#FFFF00":
		name = yellow(name)
	case "#1E90FF", "#00FFFF":
		name = cyan(name)
	}
	p = p + " " + name
	return p
}

func main() {
	iniflags.Parse()
	go ConsoleInput()
	Connect()
	channels = make(map[string]int)
	g_levelId = make(map[int]int)
	g_userName = make(map[int]string)
	g_level = make(map[int]string)
	splitchannel := strings.Split(channellist, ",")
	for i := range splitchannel {
		channels[splitchannel[i]] = i + 1
	}
	fmt.Fprintf(conn, "USER %s 8 * :%s\r\n", nick, nick)
	fmt.Fprintf(conn, "PASS %s\r\n", oauth)
	fmt.Fprintf(conn, "NICK %s\r\n", nick)
	fmt.Fprintf(conn, "CAP REQ :twitch.tv/membership twitch.tv/tags twitch.tv/commands\r\n")

	fmt.Printf("Channels: ")
	//Looping through all the channels
	for k, i := range channels {
		fmt.Fprintf(conn, "JOIN %s\r\n", k)
		fmt.Printf("#%d: %s, ", i, blue(k))
	}
	fmt.Printf("\nInserted information to server...\n")
	//Initialize DB = Create tables & add streamers
	InitDB()
	defer conn.Close()

	reader2 := bufio.NewReader(conn)
	tp := textproto.NewReader(reader2)
	go ConsoleInput()
	for {
		line, err := tp.ReadLine()
		if err != nil {
			break // break loop on errors
		}
		var (
			username string
			irc      map[string]string
			tags     map[string]string
			isTags   bool
		)
		irc = parseIRC(line)
		if irc["tags"] != "" {
			tags = parseTags(irc["tags"])
			isTags = true
		}
		go logIRC(irc)
		switch irc["command"] {
		case "PING":
			fmt.Fprintf(conn, "PONG %s\r\n", irc["trailing"])
			if debug {
				fmt.Printf(info("PONG\n"))
			}
		case "PRIVMSG":
			if isTags {
				username = tags["display-name"]
			}
			if username == "" {
				split := strings.Split(irc["prefix"], "!")
				username = strings.Replace(split[0], ":", "", 1)
			}
			msg := strings.Replace(irc["trailing"], ":", "", 1)
			fmt.Printf("[%s] %s <%s> %s\n", time.Now().Format("15:04"), blue(irc["params"]), fmtName(tags["@color"], username, tags["subscriber"], tags["turbo"], tags["user-type"]), white(msg))
			go CmdInterpreter(irc["params"], username, msg)
			//fmt.Printf("%q\n", irc)
		default:
			if debug {
				fmt.Printf("%q\n", irc)
			}
		}
	}
}

func parseIRC(l string) map[string]string {
	var k string
	s := strings.Split(l, " ")
	m := make(map[string]string)
	for i := range s {
		if k == "trailing" {
		} else if k == "" && strings.HasPrefix(s[i], "@") {
			k = "tags"
		} else if k == "" && strings.HasPrefix(s[i], ":") || k == "tags" && strings.HasPrefix(s[i], ":") {
			k = "prefix"
		} else if k == "" || k == "tags" || k == "prefix" {
			k = "command"
		} else if k == "params" && strings.HasPrefix(s[i], ":") {
			k = "trailing"
		} else {
			k = "params"
		}
		if m[k] != "" {
			m[k] = m[k] + " "
		}
		m[k] = m[k] + s[i]
	}
	return m
}

func parseTags(l string) map[string]string {
	s := strings.Split(l, ";")
	t := make(map[string]string)
	for i := range s {
		r := strings.SplitN(s[i], "=", 2)
		t[r[0]] = r[1]
	}
	return t
}

func logIRC(irc map[string]string) {
	if _, err := os.Stat("./logs"); err != nil {
		if os.IsNotExist(err) {
			color.Green("Creating directory logs")
			os.Mkdir("./logs", 0766)
		}
	}
	if strings.HasPrefix(irc["params"], "#") {
		s := strings.Split(irc["params"], " ")
		f, err := os.OpenFile("logs/"+s[0]+".log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening logfile for %s: %v", irc["params"], err)
		}
		log.SetOutput(f)
		log.Printf("%s %s %s %s\n", irc["prefix"], irc["command"], irc["params"], irc["trailing"])
		defer f.Close()
	} else {
		f, err := os.OpenFile("logs/all.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening logfile all: %v", err)
		}
		log.SetOutput(f)
		log.Printf("%s %s %s %s %s\n", irc["tags"], irc["prefix"], irc["command"], irc["params"], irc["trailing"])
		defer f.Close()
	}
}
