package main

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"strings"
	"time"
	"flag"
	"github.com/fatih/color"
	"github.com/vharitonsky/iniflags"
)

var (
    red = color.New(color.FgRed).SprintFunc()
    green = color.New(color.FgGreen).SprintFunc()
    yellow = color.New(color.FgYellow).SprintFunc()
    blue = color.New(color.FgBlue).SprintFunc()
    magenta = color.New(color.FgMagenta).SprintFunc()
    cyan = color.New(color.FgCyan).SprintFunc()
    white = color.New(color.FgWhite).SprintFunc()
    info = color.New(color.FgWhite, color.BgGreen).SprintFunc()
    server string
    port string
    nick string
    channellist string
    database string
    oauth string
    lastmsg int64 = 0
    maxMsgTime int64 = 5
    g_levelId map[int]int
    g_userName map[int]string
    g_level map[int]string
    channels map[string]int
    conn net.Conn
)

func init() {
    flag.StringVar(&server, "server", "irc.twitch.tv", "IRC server address")
    flag.StringVar(&port, "port", "6667", "IRC server port")
    flag.StringVar(&nick, "nick", "Botname", "Bot's nickname")
    flag.StringVar(&channellist, "channellist", "#botname", "Comma separated list of channel to join")
    flag.StringVar(&database, "database", "username:password@protocol(address)/dbname?param=value", "MySQL Data Source Name")
    flag.StringVar(&oauth, "oauth", "oauth:token", "OAuth token for login, https://twitchapps.com/tmi/")
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
		fmt.Printf("[%s] %s <%s> %s\n", time.Now().Format("15:04"), blue(channel), nick, white(message))
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

func getColor(c string, name string) string {
	switch(c) {
		case "#0000FF", "#5F9EA0":
			return blue(name)
		case "#FF0000", "#B22222", "#FF7F50", "#CC0000":
			return red(name)
		case "#8A2BE2", "#FF69B4", "#FF6BB5":
			return magenta(name)
		case "#008000", "#00FF7F", "#2E8B57", "#9ACD32":
			return green(name)
		case "#DAA520", "#FF4500", "#D2691E", "#FFFF00":
			return yellow(name)
		case "#1E90FF", "#00FFFF":
			return cyan(name)
		default:
			return white(name)
	}
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
		b := i
		b++
		channels[splitchannel[i]] = b
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
		if strings.Contains(line, "PING") {
			//Twitch always gives PING tmi.twitch.tv, may change this to read the PING line
			fmt.Fprintf(conn, "PONG tmi.twitch.tv\r\n")
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
				go CmdInterpreter(chanmsg[0], username, chanmsg[1])
			}
		}
	}
}
