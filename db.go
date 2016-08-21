// db
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/matryer/try.v1"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var db *sql.DB
var dberr error
var v map[string]interface{}
var k int64

//InitDB initializes mysql connection and creates tables if those doesn't exist, adds joined channels to Streamers table and assign the StreamerID
func InitDB() {
	db, dberr = sql.Open("mysql", database)
	if dberr != nil {
		log.Fatalf("Error on initializing database connection: %s", dberr.Error())
	}
	//[MySQL] packets.go:118: write unix /var/lib/mysql/mysql.sock: broken pipe
	db.SetMaxIdleConns(0)

	//Create tables
	_, dberr = db.Exec("CREATE TABLE IF NOT EXISTS Streamers ( StreamID MEDIUMINT NOT NULL, Name VARCHAR(25) NOT NULL UNIQUE, PRIMARY KEY (StreamID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
	if dberr != nil {
		log.Fatalf("Error on initializing table Streamers: %s", dberr.Error())
	}
	_, dberr = db.Exec("CREATE TABLE IF NOT EXISTS Levels ( LevelID MEDIUMINT NOT NULL AUTO_INCREMENT, StreamID MEDIUMINT NOT NULL, Nick VARCHAR(25) NOT NULL, Level VARCHAR(22) NOT NULL, Message VARCHAR(255) NOT NULL, Comment VARCHAR(255) NOT NULL, Played BOOLEAN NOT NULL, Skipped BOOLEAN NOT NULL, Added DATETIME NOT NULL, Passed DATETIME NOT NULL, Removed BOOLEAN NOT NULL, Title VARCHAR(100) NOT NULL, Difficulty TINYINT NOT NULL, Style TINYINT NOT NULL, Creator VARCHAR(50) NOT NULL, Flag VARCHAR(2) NOT NULL, Created DATE NOT NULL, Tags VARCHAR(13) NOT NULL, Image VARCHAR(100) NOT NULL, ImageFull VARCHAR(100) NOT NULL, PRIMARY KEY (LevelID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
	if dberr != nil {
		log.Fatalf("Error on initializing table Levels: %s", dberr.Error())
	}
	_, dberr = db.Exec("CREATE TABLE IF NOT EXISTS Subscribers ( SubID MEDIUMINT NOT NULL AUTO_INCREMENT, StreamID MEDIUMINT NOT NULL, Nick VARCHAR(25) NOT NULL, MonthsInRow TINYINT NOT NULL, MonthsTotal TINYINT NOT NULL, Lastsub DATETIME NOT NULL,PRIMARY KEY (SubID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
	if dberr != nil {
		log.Fatalf("Error on initializing table Subscribers: %s", dberr.Error())
	}

	blue := color.New(color.FgBlue).SprintFunc()
	var Streamer int
	fmt.Printf("dbStreamers: ")
	for k, i := range channels {
		chanName := strings.Replace(k, "#", "", 1)
		checkStream := db.QueryRow("SELECT StreamID FROM Streamers WHERE Name=?;", chanName).Scan(&Streamer)
		switch {
		case checkStream == sql.ErrNoRows:
			color.Yellow("No streamer ID, Adding...\n")
			insertStream, dberr := db.Prepare("INSERT Streamers SET Name=?,StreamID=?;")
			if dberr != nil {
				log.Fatalf("Cannot prepare streamer %s, error: %s\n", chanName, dberr.Error())
			}
			defer insertStream.Close()
			execStream, dberr := insertStream.Exec(chanName, i)
			if dberr != nil {
				log.Fatalf("Cannot add streamer %s, error: %s\n", chanName, dberr.Error())
			}
			lastID, dberr := execStream.LastInsertId()
			if dberr != nil {
				log.Fatalf("LastID error with streamer %s, error: %s\n", chanName, dberr.Error())
			}
			color.Green("New streamID for %s is #%d, ID = %d\n", k, i, lastID)
		case checkStream != nil:
			log.Fatalf("Database query to Streamers table error: %s\n", checkStream.Error())
		default:
			fmt.Printf("#%d: %s, ", Streamer, blue(k))
		}
	}
	fmt.Printf("\n")
}

func writeLevelDB(channel string, userName string, userMessage string, levelID string) {
	chanID := channels[channel]
	//Check for duplicate LevelId for this channel
	var duplicateLevel string
	var info map[string]string
	var exist bool
	info = make(map[string]string)
	err := try.Do(func(attempt int) (bool, error) {
		var err error
		info, exist, err = fetchInfo(levelID)
		return attempt < 5, err // try 5 times
	})
	if err != nil {
		log.Println("Error: " + err.Error())
	} else if exist {
		checkDuplicate := db.QueryRow("SELECT Level FROM Levels WHERE Level=? AND StreamID=?;", levelID, chanID).Scan(&duplicateLevel)
		switch {
		case checkDuplicate == sql.ErrNoRows:
			color.Green("No such level, Adding...\n")
			insertLevel, dberr := db.Prepare("INSERT Levels SET StreamID=?,Nick=?,Level=?,Message=?,Added=?,Removed=?,Title=?,Difficulty=?,Style=?,Creator=?,Flag=?,Created=?,Tags=?,Image=?,ImageFull=?;")
			if dberr != nil {
				log.Fatalf("Cannot prepare insertLevel on %s: %s\n", channel, dberr.Error())
			}
			defer insertLevel.Close()
			timeNow := time.Now().Format(time.RFC3339)
			execLevel, dberr := insertLevel.Exec(chanID, userName, levelID, userMessage, timeNow, 0, info["title"], info["diff"], info["style"], info["name"], info["flag"], info["created"], info["tags"], info["img"], info["imgfull"])
			if dberr != nil {
				log.Fatalf("Cannot exec insertLevel on %s: %s\n", channel, dberr.Error())
			}
			rowsAff, dberr := execLevel.RowsAffected()
			if dberr != nil {
				log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
			}
			lastID, dberr := execLevel.LastInsertId()
			if dberr != nil {
				log.Fatalf("No lastID on %s: %s\n", channel, dberr.Error())
			}
			color.Green("Added level %s by %s for %d %s. Row|#: %d|%d\n", levelID, userName, chanID, channel, rowsAff, lastID)
		case checkDuplicate != nil:
			log.Fatalf("Checking duplicate level failed, error: %s\n", checkDuplicate.Error())
		default:
			color.Yellow("Duplicate level, not adding...\n")
		}
	} else {
		color.Yellow("Level doesn't exist...\n")
	}
}

func getLevel(streamer bool, channel string, comment string) (result string) {
	var online bool
	chanID := channels[channel]
	//Choose new random level if streamer, else get last random level
	if streamer {
		if gLevelID[chanID] != 0 && comment != "" {
			doComment(comment, gLevelID[chanID])
		}
		var levelID int
		var userName string
		var level string
		var message string
		var title string
		var diff int
		var style int
		var creator string
		var flag string
		var removed int
		var tags string
		getrLevel, dberr := db.Query("SELECT LevelID,Nick,Level,Message,Removed,Title,Difficulty,Style,Creator,Flag,Tags FROM Levels WHERE Played=0 AND StreamID=? ORDER BY RAND() LIMIT 100;", chanID)
		if dberr == sql.ErrNoRows {
			return "No unplayed levels in database"
		}
		if dberr != nil {
			log.Fatalf("Cannot get random level: %s\n", dberr.Error())
		}
		for getrLevel.Next() {
			dberr = getrLevel.Scan(&levelID, &userName, &level, &message, &removed, &title, &diff, &style, &creator, &flag, &tags)
			fmt.Printf("#%d %s by %s | ", levelID, level, userName)
			if removed == 1 {
				color.Red("Removed level, skipping")
				continue
			}
			var o bool
			err := try.Do(func(attempt int) (bool, error) {
				var err error
				o, err = isWatching(channel, userName)
				return attempt < 5, err // try 5 times
			})
			if err != nil {
				log.Println("Try error:", err)
			}
			if o {
				gLevelID[chanID] = levelID
				gUserName[chanID] = userName
				gLevel[chanID] = level
				color.Green("Online\n")
				online = true
				break
			} else {
				color.Red("Offline\n")
			}
		}
		defer getrLevel.Close()
		if getrLevel.Next() == false && online == false {
			//color.Red("No online level, RIP\n")
			return "No unplayed levels or online submitters, try again"
		}
		updatePlayed, dberr := db.Prepare("UPDATE Levels SET Played=1,Passed=? WHERE LevelID=?;")
		if dberr != nil {
			log.Fatalf("Cannot prepare updatePlayed on %s: %s\n", channel, dberr.Error())
		}
		timeNow := time.Now().Format(time.RFC3339)
		execPlayed, dberr := updatePlayed.Exec(timeNow, gLevelID[chanID])
		if dberr != nil {
			log.Fatalf("Cannot exec updatePlayed on %s: %s\n", channel, dberr.Error())
		}
		rowsAff, dberr := execPlayed.RowsAffected()
		if dberr != nil {
			log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
		}
		fmt.Printf("Updated played=true for level %d, rows %d\n", gLevelID[chanID], rowsAff)
		chanName := strings.Replace(channel, "#", "", 1)
		msg := strings.Replace(message, "%", "%%", -1)
		if tags != "" {
			tags = "|" + tags
		}
		result = fmt.Sprintf("%s: %s | %s [%s|%s%s] by %s [%s] | <%s> %s", chanName, gLevel[chanID], title, getDifficulty(diff), getStyle(style), tags, creator, flag, gUserName[chanID], msg)
	} else {
		if gLevel[chanID] == "" {
			result = "Level not selected BibleThump"
		} else {
			result = fmt.Sprintf("Last played level #%d: %s by %s", gLevelID[chanID], gLevel[chanID], gUserName[chanID])
			//return result
		}
	}
	return
}

func getDifficulty(diff int) (t string) {
	switch diff {
	case 0:
		t = "N/A"
	case 1:
		t = "Easy"
	case 2:
		t = "Normal"
	case 3:
		t = "Expert"
	case 4:
		t = "Super Expert"
	}
	return
}

func getStyle(style int) (t string) {
	switch style {
	case 0:
		t = "N/A"
	case 1:
		t = "SMB"
	case 2:
		t = "SMB3"
	case 3:
		t = "SMW"
	case 4:
		t = "NSMBU"
	}
	return
}

func doReroll(channel string) (result string) {
	chanID := channels[channel]
	if gLevel[chanID] == "" {
		result = "Cannot reroll without level Kappa"
	} else {
		//Save old levelId and get new level before setting Played back to false
		oldLevelID := gLevelID[chanID]
		result = getLevel(true, channel, "")
		rerollPlayed, dberr := db.Prepare("UPDATE Levels SET Played=0,Passed='0000-00-00 00:00:00' WHERE LevelID=?;")
		if dberr != nil {
			log.Fatalf("Cannot revert rerollPlayed on %s: %s\n", channel, dberr.Error())
		}
		execrPlayed, dberr := rerollPlayed.Exec(oldLevelID)
		if dberr != nil {
			log.Fatalf("Cannot exec rerollPlayed on %s: %s\n", channel, dberr.Error())
		}
		rowsAff, dberr := execrPlayed.RowsAffected()
		if dberr != nil {
			log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
		}
		fmt.Printf("Updated played=false for level %d , rows affected %d\n", oldLevelID, rowsAff)
	}
	return
}

func doSkip(channel string, comment string) (result string) {
	chanID := channels[channel]
	if gLevel[chanID] == "" {
		result = "Cannot skip without level Kappa"
	} else {
		//Save old levelId and get new level before setting Played back to false
		oldLevelID := gLevelID[chanID]
		result = getLevel(true, channel, comment)
		skipPlayed, dberr := db.Prepare("UPDATE Levels SET Skipped=1 WHERE LevelID=?;")
		if dberr != nil {
			log.Fatalf("Cannot skip skipPlayed on %s: %s\n", channel, dberr.Error())
		}
		execPlayed, dberr := skipPlayed.Exec(oldLevelID)
		if dberr != nil {
			log.Fatalf("Cannot exec skipPlayed on %s: %s\n", channel, dberr.Error())
		}
		rowsAff, dberr := execPlayed.RowsAffected()
		if dberr != nil {
			log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
		}
		fmt.Printf("Updated skipped=true for level %d , rows affected %d\n", oldLevelID, rowsAff)
	}
	return
}

func doComment(comment string, levelID int) {
	addComment, dberr := db.Prepare("UPDATE Levels SET Comment=? WHERE LevelID=?;")
	if dberr != nil {
		log.Fatalf("Cannot add comment on %d: %s\n", levelID, dberr.Error())
	}
	execComment, dberr := addComment.Exec(comment, levelID)
	if dberr != nil {
		log.Fatalf("Cannot exec addComment on %d: %s\n", levelID, dberr.Error())
	}
	rowsAff, dberr := execComment.RowsAffected()
	if dberr != nil {
		log.Fatalf("No rows changed on %d: %s\n", levelID, dberr.Error())
	}
	fmt.Printf("Added comment for level %d , rows affected %d\n", levelID, rowsAff)
}

func getStats(channel string) string {

	chanID := channels[channel]

	var allCount int
	var playCount int
	var skipCount int
	allLevels := db.QueryRow("SELECT count(Played) FROM Levels WHERE StreamID=?;", chanID).Scan(&allCount)
	if allLevels != nil {
		log.Fatalf("Cannot count levels: %s", allLevels.Error())
	}
	playedLevels := db.QueryRow("SELECT count(Played) FROM Levels WHERE StreamID=? AND Played=1 AND Skipped=0;", chanID).Scan(&playCount)
	if playedLevels != nil {
		log.Fatalf("Cannot count played levels: %s", playedLevels.Error())
	}
	skipLevels := db.QueryRow("SELECT count(Played) FROM Levels WHERE StreamID=? AND Skipped=1;", chanID).Scan(&skipCount)
	if skipLevels != nil {
		log.Fatalf("Cannot count skipped levels: %s", skipLevels.Error())
	}
	result := fmt.Sprintf("Streamer has %d lvls played and %d lvls skipped out of %d levels", playCount, skipCount, allCount)
	return result
}

func isWatching(channel string, name string) (bool, error) {
	chanName := strings.Replace(channel, "#", "", 1)
	if k+60 <= time.Now().Unix() {
		var err error
		url := "http://tmi.twitch.tv/group/user/" + chanName + "/chatters"
		response, err := http.Get(url)
		if err != nil {
			log.Printf("Cannot get URL response: %s\n", err.Error())
			return false, err
		}
		defer response.Body.Close()
		dec := json.NewDecoder(response.Body)
		if err := dec.Decode(&v); err != nil {
			log.Printf("Parse error: %s\n", err.Error())
			return false, err
		}
		k = time.Now().Unix()
		color.Yellow("Updating chatters")
	}
	//fmt.Printf("%q\n", v)
	chats := v["chatters"].(map[string]interface{})
	views := chats["viewers"].([]interface{})
	mods := chats["moderators"].([]interface{})
	for _, b := range views {
		if b == strings.ToLower(name) {
			return true, nil
		}
	}
	for _, b := range mods {
		if b == strings.ToLower(name) {
			return true, nil
		}
	}
	return false, nil
}

func writeSubs(channel string, name string, months string) {
	chanID := channels[channel]
	emote := map[int]string{0: "rtqPilu rtqKeppi", 1: "rtqPeli", 2: "rtqTroll", 3: "rtqMega", 4: "rtqBoser", 5: "rtqKemu", 6: "rtqKinder", 7: "rtqJano"}
	var monthsTotal int
	var newTotal int
	var subID int
	monthsInt, err := strconv.Atoi(months)
	if err != nil {
		log.Fatalf("Error converting months to monthsInt: %s", err.Error())
	}
	checkSub := db.QueryRow("SELECT SubID,MonthsTotal FROM Subscribers WHERE Nick=? AND StreamID=?;", name, chanID).Scan(&subID, &monthsTotal)
	switch {
	case checkSub == sql.ErrNoRows:
		color.Green("No such subscriber, Adding...\n")
		insertSub, dberr := db.Prepare("INSERT Subscribers SET StreamID=?,Nick=?,MonthsInRow=?,MonthsTotal=?,Lastsub=?;")
		if dberr != nil {
			log.Fatalf("Cannot prepare insertSub on %s: %s\n", channel, dberr.Error())
		}
		defer insertSub.Close()
		timeNow := time.Now().Format(time.RFC3339)
		execSub, dberr := insertSub.Exec(chanID, name, months, months, timeNow)
		if dberr != nil {
			log.Fatalf("Cannot exec insertSub on %s: %s\n", channel, dberr.Error())
		}
		rowsAff, dberr := execSub.RowsAffected()
		if dberr != nil {
			log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
		}
		color.Green("Added Sub %s for %s months on %s, %d\n", name, months, channel, rowsAff)
		if channel == "#retku" {
			var msg string
			k := getSubs(chanID)
			if monthsInt == 1 {
				msg = fmt.Sprintf("Tervetuloa Retkueeseen %s %s Päivän %d. subi!", name, GetRand(emote), k)
			} else {
				msg = fmt.Sprintf("%s kuukautta putkeen Retkueessa %s %s Päivän %d. subi!", months, name, GetRand(emote), k)
			}
			Message(channel, msg)
		}
	case checkSub != nil:
		log.Fatalf("Checking for subs failed, error: %s\n", checkSub.Error())
	default:
		updateSub, dberr := db.Prepare("UPDATE Subscribers SET MonthsInRow=?,MonthsTotal=?,Lastsub=? WHERE SubID=?;")
		if dberr != nil {
			log.Fatalf("Cannot prepare updateSub on %s: %s\n", channel, dberr.Error())
		}
		if monthsInt > monthsTotal+1 {
			newTotal = monthsInt
		} else {
			newTotal = monthsTotal + 1
		}
		timeNow := time.Now().Format(time.RFC3339)
		execSubU, dberr := updateSub.Exec(months, newTotal, timeNow, subID)
		if dberr != nil {
			log.Fatalf("Cannot exec updateSub on %s: %s\n", channel, dberr.Error())
		}
		rowsAff, dberr := execSubU.RowsAffected()
		if dberr != nil {
			log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
		}
		color.Green("Updated sub %s for %s months and %d total months on %s, %d\n", name, months, newTotal, channel, rowsAff)
		if channel == "#retku" {
			var msg string
			k := getSubs(chanID)
			if monthsInt == 1 {
				msg = fmt.Sprintf("Tervetuloa takaisin Retkueeseen %s %s , laskujeni mukaan yhteensä %d kuukautta Retkueessa! Päivän %d. subi!", name, GetRand(emote), newTotal, k)
			} else {
				msg = fmt.Sprintf("%s kuukautta putkeen Retkueessa %s %s Päivän %d. subi!", months, name, GetRand(emote), k)
				//tarkasta onko months ja total sama
			}
			Message(channel, msg)
		}
	}
}

func getSubs(chanID int) int {
	var i int
	subToday := db.QueryRow("SELECT count(*) FROM Subscribers WHERE StreamID=? AND Lastsub>=CURDATE();", chanID).Scan(&i)
	if subToday != nil {
		log.Fatalf("Cannot count subs: %s", subToday.Error())
	}
	return i
}

//GetRand gets random string from map[int]string
func GetRand(a map[int]string) string {
	// produce a pseudo-random number between 0 and len(a)-1
	i := int(float32(len(a)) * rand.Float32())
	for _, v := range a {
		if i == 0 {
			return v
		} else {
			i--
		}
	}
	panic("impossible")
}

//Fetch information from level
func fetchInfo(levelID string) (info map[string]string, exist bool, err error) {
	info = make(map[string]string)
	url := "https://supermariomakerbookmark.nintendo.net/courses/" + strings.ToUpper(levelID)
	r, err := goquery.NewDocument(url)
	if err != nil {
		log.Printf("Cannot get URL response: %s\n", err.Error())
		return
	}
	//This should work for now, maybe later check the http header for errorcodes
	ecode := typography(r, ".error-code")
	switch ecode {
	case "404":
		fmt.Printf("%s: Not found\n", ecode)
		exist = false
	case "500", "502":
		fmt.Printf("%s: Server error\n", ecode)
		err = fmt.Errorf("%s: Server error\n", ecode)
	default:
		info["title"] = r.Find(".course-title").Text()
		info["diff"] = strings.TrimSpace(r.Find(".course-header").Text())
		if len(info["diff"]) == 0 {
			info["diff"] = "0"
			info["clear"] = ""
			info["cleartimef"] = ""
		} else {
			switch info["diff"] {
			case "Easy":
				info["diff"] = "1"
			case "Normal":
				info["diff"] = "2"
			case "Expert":
				info["diff"] = "3"
			case "Super Expert":
				info["diff"] = "4"
			}
			info["clear"] = typography(r, ".clear-rate")
			info["cleartime"] = typography(r, ".clear-time")
		}
		gameskin, _ := r.Find(".gameskin").Attr("class")
		switch gameskin {
		case "gameskin bg-image common_gs_sb":
			info["style"] = "1"
		case "gameskin bg-image common_gs_sb3":
			info["style"] = "2"
		case "gameskin bg-image common_gs_sw":
			info["style"] = "3"
		case "gameskin bg-image common_gs_sbu":
			info["style"] = "4"
		}
		c := r.Find(".created_at").Text()
		if strings.Contains(c, "ago") == true {
			s := strings.Split(c, " ")
			cr, err := strconv.Atoi(s[0])
			if err != nil {
				log.Printf(err.Error())
			}
			switch s[1] {
			case "hour", "hours":
				info["created"] = time.Now().Add(time.Duration(-cr) * time.Hour).Format(time.RFC3339)
			case "day", "days":
				info["created"] = time.Now().AddDate(0, 0, -cr).Format(time.RFC3339)
			}
		} else {
			parsed, err := time.Parse("01/02/2006", c)
			if err != nil {
				log.Printf(err.Error())
			}
			info["created"] = parsed.Format(time.RFC3339)
		}

		info["liked"] = typography(r, ".liked-count")
		info["played"] = typography(r, ".played-count")
		info["shared"] = typography(r, ".shared-count")
		info["tried"] = typography(r, ".tried-count")
		info["tags"] = r.Find(".course-meta-info .course-tag").Text()
		if info["tags"] == "---" {
			info["tags"] = ""
		}
		flagRaw, _ := r.Find(".flag").Attr("class")
		info["flag"] = strings.Replace(flagRaw, "flag ", "", 1)
		info["name"] = r.Find(".creator-info .name").Text()
		info["img"], _ = r.Find("img.course-image").Attr("src")
		info["imgfull"], _ = r.Find(".course-image-full").Attr("src")

		//info = fmt.Sprintf("%s [%s|%s] by %s [%s], Created: %s, Tags: %s, Clear: %s (%s) WR|%s, Likes|Played|Shared: %s|%s|%s, Images: %s %s", course, diff, style, name, flag, created, tags, clear, tried, cleartime, liked, played, shared, image, imagefull)
		exist = true
	}
	return info, exist, err
}

//Convert typography classes to numberstring
func typography(r *goquery.Document, class string) (numbers string) {
	r.Find(class + " .typography").Each(func(i int, s *goquery.Selection) {
		typo, _ := s.Attr("class")
		number := strings.Replace(typo, "typography typography-", "", 1)
		switch number {
		case "second":
			number = "."
		case "percent":
			number = "%"
		case "slash":
			number = "/"
		case "minute":
			number = ":"
		}
		numbers = numbers + number
	})
	return
}
