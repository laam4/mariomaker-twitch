// db
package main

import (
	"log"
	"time"
	"fmt"
	"strings"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/fatih/color"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

var db *sql.DB
var dberr error

func InitDB() {
        db, dberr = sql.Open("mysql", database)
        if dberr != nil {
                log.Fatalf("Error on initializing database connection: %s", dberr.Error())
        }

        _, dberr = db.Exec("CREATE TABLE IF NOT EXISTS Streamers ( StreamID MEDIUMINT NOT NULL, Name VARCHAR(25) NOT NULL UNIQUE, PRIMARY KEY (StreamID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
        if dberr != nil {
                log.Fatalf("Error on initializing table Streamers: %s", dberr.Error())
        }
        _, dberr = db.Exec("CREATE TABLE IF NOT EXISTS Levels ( LevelID MEDIUMINT NOT NULL AUTO_INCREMENT, StreamID MEDIUMINT NOT NULL, Nick VARCHAR(25) NOT NULL, Level VARCHAR(22) NOT NULL, Message VARCHAR(255) NOT NULL, Comment VARCHAR(255) NOT NULL, Played BOOLEAN NOT NULL, Skipped BOOLEAN NOT NULL, Added DATETIME NOT NULL, Passed DATETIME NOT NULL,PRIMARY KEY (LevelID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
        if dberr != nil {
                log.Fatalf("Error on initializing table Levels: %s", dberr.Error())
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
                        lastId, dberr := execStream.LastInsertId()
                        if dberr != nil {
                                log.Fatalf("Last id error with streamer %s, error: %s\n", chanName, dberr.Error())
                        }
                        color.Green("New streamId for %s is #%d, ID = %d\n", k, i, lastId)
                case checkStream != nil:
                        log.Fatalf("Database query to Streamers table error: %s\n", checkStream.Error())
                default:
			fmt.Printf("#%d: %s, ", Streamer, blue(k))
                }
        }
	fmt.Printf("\n")
}

func writeLevelDB(channel string, userName string, userMessage string, levelId string) {
	chanId := channels[channel]
//Check for duplicate LevelId for this channel
        var duplicateLevel string
        checkDuplicate := db.QueryRow("SELECT Level FROM Levels WHERE Level=? AND StreamID=?;", levelId,chanId).Scan(&duplicateLevel)
        switch {
        case checkDuplicate == sql.ErrNoRows:
                color.Green("No such level, Adding...\n")
                insertLevel, dberr := db.Prepare("INSERT Levels SET StreamID=?,Nick=?,Level=?,Message=?,Added=?;")
		if dberr != nil {
			log.Fatalf("Cannot prepare insertLevel on %s: %s\n", channel, dberr.Error())
		}
		defer insertLevel.Close()
                timeNow := time.Now().Format(time.RFC3339)
                execLevel, dberr := insertLevel.Exec(chanId, userName, levelId, userMessage, timeNow)
		if dberr != nil {
			log.Fatalf("Cannot exec insertLevel on %s: %s\n", channel, dberr.Error())
		}
                rowsAff, dberr := execLevel.RowsAffected()
		if dberr != nil {
			log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
		}
		lastId, dberr := execLevel.LastInsertId()
                if dberr != nil {
                        log.Fatalf("No last id on %s: %s\n", channel, dberr.Error())
                }
		color.Green("Added level %s by %s for %d %s. Row|#: %d|%d\n", levelId, userName, chanId, channel, rowsAff, lastId)
        case checkDuplicate != nil:
                log.Fatalf("Checking duplicate level failed, error: %s\n", checkDuplicate.Error())
        default:
                color.Yellow("Duplicate level, not adding...\n")
        }
}

func getLevel(streamer bool, channel string, comment string) string {
	var result string
	var online bool
	chanId := channels[channel]
	//Choose new random level if streamer, else get last random level
	if streamer {
		if g_levelId[chanId] != 0 && comment != "" {
			doComment(comment, g_levelId[chanId])
                }
		var levelId int
		var userName string
		var level string
		var message string
        	var added string
		getrLevel, dberr := db.Query("SELECT LevelID,Nick,Level,Message,Added FROM Levels WHERE Played=0 AND StreamID=? ORDER BY RAND() LIMIT 10;", chanId)
        	if dberr == sql.ErrNoRows {
			return "No unplayed levels in database"
		}
        	if dberr != nil {
                	log.Fatalf("Cannot get random level: %s\n", dberr.Error())
		}
		for getrLevel.Next() {
                	dberr = getrLevel.Scan(&levelId,  &userName, &level, &message,  &added)
			fmt.Printf("#%d %s by %s | ", levelId, level, userName)
			if isWatching(channel,userName) {
                       		g_levelId[chanId] = levelId
                       		g_userName[chanId] = userName
                       		g_level[chanId] = level
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
			return "No submitters online for 10 random levels, try again"
		}
	        updatePlayed, dberr := db.Prepare("UPDATE Levels SET Played=1,Passed=? WHERE LevelID=?;")
       		if dberr != nil {
               		log.Fatalf("Cannot prepare updatePlayed on %s: %s\n", channel, dberr.Error())
               	}
               	timeNow := time.Now().Format(time.RFC3339)
               	execPlayed, dberr := updatePlayed.Exec(timeNow, g_levelId[chanId])
               	if dberr != nil {
               		log.Fatalf("Cannot exec updatePlayed on %s: %s\n", channel, dberr.Error())
               	}
               	rowsAff, dberr := execPlayed.RowsAffected()
               	if dberr != nil {
               		log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
               	}
               	fmt.Printf("Updated played=true for level %d, rows %d\n", g_levelId[chanId], rowsAff)
               	chanName := strings.Replace(channel, "#", "@", 1)
               	msg := strings.Replace(message, "%", "%%", -1)
               	result = fmt.Sprintf("%s: %s by %s | #%d[%s] %s", chanName, g_level[chanId], g_userName[chanId], g_levelId[chanId], added, msg)
        } else {
                if g_level[chanId] == "" {
                        return "Level not selected BibleThump"
                } else {
                result = fmt.Sprintf("Last played level #%d: %s by %s", g_levelId[chanId], g_level[chanId], g_userName[chanId])
                //return result
                }
        }
        return result
}

func doReroll(channel string) string {

        chanId := channels[channel]
        if g_level[chanId] == "" {
		return "Cannot reroll without level Kappa"
        } else {
		//Save old levelId and get new level before setting Played back to false
		oldLevelId := g_levelId[chanId]
		result := getLevel(true,channel,"")
		rerollPlayed, dberr := db.Prepare("UPDATE Levels SET Played=0,Passed='0000-00-00 00:00:00' WHERE LevelID=?;")
                if dberr != nil {
                        log.Fatalf("Cannot revert rerollPlayed on %s: %s\n", channel, dberr.Error())
                }
                execrPlayed, dberr := rerollPlayed.Exec(oldLevelId)
                if dberr != nil {
                        log.Fatalf("Cannot exec rerollPlayed on %s: %s\n", channel, dberr.Error())
                }
                rowsAff, dberr := execrPlayed.RowsAffected()
                if dberr != nil {
                        log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
                }
                fmt.Printf("Updated played=false for level %d , rows affected %d\n", oldLevelId, rowsAff)
		return result
        }
	return "Kappa"
}

func doSkip(channel string, comment string) string {

        chanId := channels[channel]
        if g_level[chanId] == "" {
                return "Cannot skip without level Kappa"
        } else {
                //Save old levelId and get new level before setting Played back to false
                oldLevelId := g_levelId[chanId]
		//if comment != "" {
		//	doComment(comment, oldLevelId)
		//}
                result := getLevel(true,channel,comment)
                skipPlayed, dberr := db.Prepare("UPDATE Levels SET Skipped=1 WHERE LevelID=?;")
                if dberr != nil {
                        log.Fatalf("Cannot skip skipPlayed on %s: %s\n", channel, dberr.Error())
                }
                execPlayed, dberr := skipPlayed.Exec(oldLevelId)
                if dberr != nil {
                        log.Fatalf("Cannot exec skipPlayed on %s: %s\n", channel, dberr.Error())
                }
                rowsAff, dberr := execPlayed.RowsAffected()
                if dberr != nil {
                        log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
                }
                fmt.Printf("Updated skipped=true for level %d , rows affected %d\n", oldLevelId, rowsAff)
                return result
        }
        return "Kappa"
}

func doComment(comment string, levelid int) {
	addComment, dberr := db.Prepare("UPDATE Levels SET Comment=? WHERE LevelID=?;")
        if dberr != nil {
		log.Fatalf("Cannot add comment on %s: %s\n", levelid, dberr.Error())
	}
	execComment, dberr := addComment.Exec(comment, levelid)
	if dberr != nil {
		log.Fatalf("Cannot exec addComment on %s: %s\n", levelid, dberr.Error())
	}
	rowsAff, dberr := execComment.RowsAffected()
	if dberr != nil {
		log.Fatalf("No rows changed on %s: %s\n", levelid, dberr.Error())
	}
	fmt.Printf("Added comment for level %d , rows affected %d\n", levelid, rowsAff)
}



func getStats(channel string) string {
	
	chanId := channels[channel]

	var allCount int
	var playCount int
	var skipCount int
	allLevels := db.QueryRow("SELECT count(Played) FROM Levels WHERE StreamID=?;", chanId).Scan(&allCount)
	if allLevels != nil {
		log.Fatalf("Cannot count levels: %s", allLevels.Error())
	}
	playedLevels := db.QueryRow("SELECT count(Played) FROM Levels WHERE StreamID=? AND Played=1 AND Skipped=0;", chanId).Scan(&playCount)
	if playedLevels != nil {
		log.Fatalf("Cannot count played levels: %s", playedLevels.Error())
	}
        skipLevels := db.QueryRow("SELECT count(Played) FROM Levels WHERE StreamID=? AND Skipped=1;", chanId).Scan(&skipCount)
        if skipLevels != nil {
                log.Fatalf("Cannot count skipped levels: %s", skipLevels.Error())
        }
	result := fmt.Sprintf("Streamer has %d lvls played and %d lvls skipped out of %d levels", playCount, skipCount, allCount)
        return result
}

func isWatching(channel string, name string) bool {
	chanName := strings.Replace(channel, "#", "", 1)
        url := "http://tmi.twitch.tv/group/user/"+chanName+"/chatters"
        response, err := http.Get(url)
        if err != nil {
                log.Fatalf("Cannot get URL response: %s\n", err.Error())
        } else {
                defer response.Body.Close()
                data, err := ioutil.ReadAll(response.Body)
                if err != nil {
                        log.Fatalf("Cannot read URL response: %s\n", err.Error())
                }
                var parsed map[string]interface{}
                p := json.Unmarshal(data, &parsed)
                if p != nil {
                        log.Fatalf("Parse error: %s\n", err.Error())
                }
                chats := parsed["chatters"].(map[string]interface{})
                views := chats["viewers"].([]interface{})
                mods := chats["moderators"].([]interface{})
                for _, b := range views {
                        if b == strings.ToLower(name) {
                                return true
                        }
                }
                for _, b := range mods {
                        if b == strings.ToLower(name) {
                                return true
                        }
                }
        }
        return false
}
