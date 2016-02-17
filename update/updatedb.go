//This is experimental and may not work and has no options, yet
//Please take backup from your database
package main

import (
	"database/sql"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/matryer/try.v1"
	"log"
	"strconv"
	"strings"
	"time"
)

var db *sql.DB
var dberr error
//false if first time running, true after first run
var updated bool = false

func InitDB() {
	database := "mario:salakala@unix(/var/run/mysqld/mysqld.sock)/mariomaker"
	db, dberr = sql.Open("mysql", database)
	if dberr != nil {
		log.Fatalf("Error on initializing database connection: %s", dberr.Error())
	}
	if !updated {
		_, dberr = db.Exec("ALTER TABLE Levels ADD ( Removed BOOLEAN NOT NULL, Title VARCHAR(100) NOT NULL, Difficulty TINYINT NOT NULL, Style TINYINT NOT NULL, Creator VARCHAR(50) NOT NULL, Flag VARCHAR(2) NOT NULL, Created DATE NOT NULL, Tags VARCHAR(13) NOT NULL, Image VARCHAR(100) NOT NULL, ImageFull VARCHAR(100) NOT NULL );")
		if dberr != nil {
			log.Fatalf("Error adding columns to Levels: %s", dberr.Error())
		}
	}
}

func main() {
	n0 := time.Now()
	InitDB()
	UpdateExistingLevels()
	n1 := time.Now()
	fmt.Printf("The whole update took %v to run.\n", n1.Sub(n0))
}

func UpdateExistingLevels() {
	var id int
	var level string
	allRows, dberr := db.Query("SELECT LevelID,Level From Levels;")
	if dberr != nil {
		log.Fatalf("Query failed: %s\n", dberr.Error())
	}
	defer allRows.Close()
	for allRows.Next() {
		t0 := time.Now()
		err := allRows.Scan(&id, &level)
		if err != nil {
			log.Fatalf("Row scan error: %s\n", dberr.Error())
		}
		var info map[string]string
		var exist bool
		info = make(map[string]string)
		tryerr := try.Do(func(attempt int) (bool, error) {
			var err error
			info, exist, err = fetchInfo(level)
			return attempt < 5, err // try 5 times
		})
		if tryerr != nil {
			log.Println("Error: " + tryerr.Error())
		} else if exist {
			color.Green("Level found, updating\n")
			updateInfo, dberr := db.Prepare("UPDATE Levels SET Removed=?,Title=?,Difficulty=?,Style=?,Creator=?,Flag=?,Created=?,Tags=?,Image=?,ImageFull=? WHERE LevelID=?;")
			if dberr != nil {
				log.Fatalf("Cannot prepare updateInfo on %s: %s\n", id, dberr.Error())
			}
			execInfo, dberr := updateInfo.Exec(0, info["title"], info["diff"], info["style"], info["name"], info["flag"], info["created"], info["tags"], info["img"], info["imgfull"], id)
			if dberr != nil {
				log.Fatalf("Cannot execute updateInfo on %s: %s\n", id, dberr.Error())
			}
			rowsAff, dberr := execInfo.RowsAffected()
			if dberr != nil {
				log.Fatalf("No rows changed on %s: %s\n", id, dberr.Error())
			}
			fmt.Printf("Rows affected %d\n", rowsAff)
		} else {
			color.Red("Level doesn't exist!\n")
			updateInfo, dberr := db.Prepare("UPDATE Levels SET Removed=? WHERE LevelID=?;")
			if dberr != nil {
				log.Fatalf("Cannot prepare updateInfo/removed on %s: %s\n", id, dberr.Error())
			}
			execInfo, dberr := updateInfo.Exec(1, id)
			if dberr != nil {
				log.Fatalf("Cannot execute updateInfo/removed on %s: %s\n", id, dberr.Error())
			}
			rowsAff, dberr := execInfo.RowsAffected()
			if dberr != nil {
				log.Fatalf("No rows changed on %s: %s\n", id, dberr.Error())
			}
			fmt.Printf("Rows affected %d\n", rowsAff)
		}
		t1 := time.Now()
		fmt.Printf("Updating level %d took %v to run.\n", id, t1.Sub(t0))
	}
	dberr = allRows.Err()
	if dberr != nil {
		log.Fatalf("Row error: %s\n", dberr.Error())
	}
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
