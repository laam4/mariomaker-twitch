// db
package main

import (
	"log"
	"time"
	"fmt"
	"strings"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB
var dberr error

func (bot *Bot) InitDB() {
        db, dberr = sql.Open("mysql", bot.user+":"+bot.pass+"@"+bot.host+"/"+bot.database)
        if dberr != nil {
                log.Fatalf("Error on initializing database connection: %s", dberr.Error())
        }

        _, dberr = db.Exec("CREATE TABLE IF NOT EXISTS Streamers ( StreamID MEDIUMINT NOT NULL, Name VARCHAR(25) NOT NULL UNIQUE, PRIMARY KEY (StreamID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
        if dberr != nil {
                log.Fatalf("Error on initializing table Streamers: %s", dberr.Error())
        }
        _, dberr = db.Exec("CREATE TABLE IF NOT EXISTS Levels ( LevelID MEDIUMINT NOT NULL AUTO_INCREMENT, StreamID MEDIUMINT NOT NULL, Nick VARCHAR(25) NOT NULL, Level VARCHAR(22) NOT NULL, Message VARCHAR(255) NOT NULL, Played BOOLEAN NOT NULL, Added DATETIME NOT NULL, Passed DATETIME NOT NULL,PRIMARY KEY (LevelID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
        if dberr != nil {
                log.Fatalf("Error on initializing table Levelsn: %s", dberr.Error())
        }

        var Streamer int
        for i := range bot.channel {
                chanName := strings.Replace(bot.channel[i], "#", "", 1)
                checkStream := db.QueryRow("SELECT StreamID FROM Streamers WHERE Name=?", chanName).Scan(&Streamer)
                switch {
                case checkStream == sql.ErrNoRows:
                        fmt.Printf("No streamer ID, Adding...\n")
                        insertStream, dberr := db.Prepare("INSERT Streamers SET Name=?,StreamID=?")
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
                        fmt.Printf("New streamId for %s is %d, ID = %d\n", bot.channel[i], i, lastId)
                case checkStream != nil:
                        log.Fatalf("Database query to Streamers table error: %s\n", checkStream.Error())
                default:
                        fmt.Printf("StreamerId for %s is %d\n", bot.channel[i], Streamer)
                }
        }
}

func (bot *Bot) writeLevelDB(channel string, userName string, userMessage string, levelId string) {
	
	chanId := bot.getChanId(channel)
	//Check for duplicate LevelId for this channel
        var duplicateLevel string
        checkDuplicate := db.QueryRow("SELECT Level FROM Levels WHERE Level=? AND StreamID=?", levelId,chanId).Scan(&duplicateLevel)
        switch {
        case checkDuplicate == sql.ErrNoRows:
                fmt.Printf("No such level, Adding...\n")
                insertLevel, dberr := db.Prepare("INSERT Levels SET StreamID=?,Nick=?,Level=?,Message=?,Added=?")
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
		fmt.Printf("Added level %s by %s for %d %s. Rows affected: %d Last ID: %d\n", levelId, userName, chanId, channel, rowsAff, lastId)
        case checkDuplicate != nil:
                log.Fatalf("Checking duplicate level failed, error: %s\n", checkDuplicate.Error())
        default:
                fmt.Printf("Duplicate level, not adding...\n")
        }
}

func (bot *Bot) getLevel(streamer bool, channel string) string {
	
	chanId := bot.getChanId(channel)
	
	//Choose new random level if streamer, else get last random level
	if streamer {
		var levelId int
		var userName string
		var level string
		var message string
        	var added string
		getrLevel := db.QueryRow("SELECT LevelID,Nick,Level,Message,Added FROM Levels WHERE Played=0 AND StreamID=? ORDER BY RAND() LIMIT 1;", chanId).Scan(&levelId,  &userName, &level, &message,  &added,)
	        switch {
        	case getrLevel == sql.ErrNoRows:
			return "No unplayed levels in database"
        	case getrLevel != nil:
                	log.Fatalf("Cannot get random level: error\n", getrLevel.Error())
        	default:
                	fmt.Printf("New random level chosen #%d %s by %s\n", bot.levelId[chanId], bot.level[chanId], bot.userName[chanId])
			bot.levelId[chanId] = levelId
			bot.userName[chanId] = userName
			bot.level[chanId] = level
        	}

                updatePlayed, dberr := db.Prepare("UPDATE Levels SET Played=1,Passed=? WHERE LevelID=?")
		if dberr != nil {
			log.Fatalf("Cannot prepare updatePlayed on %s: %s\n", channel, dberr.Error())
		}
		timeNow := time.Now().Format(time.RFC3339)
                execPlayed, dberr := updatePlayed.Exec(timeNow, bot.levelId[chanId])
		if dberr != nil {
			log.Fatalf("Cannot exec updatePlayed on %s: %s\n", channel, dberr.Error())
		}
                rowsAff, dberr := execPlayed.RowsAffected()
		if dberr != nil {
			log.Fatalf("No rows changed on %s: %s\n", channel, dberr.Error())
		}
                fmt.Printf("Updated played=true for level %d , rows affected %d\n", bot.levelId[chanId], rowsAff)
		chanName := strings.Replace(channel, "#", "@", 1)
		result := fmt.Sprintf("%s: #%d %s by %s | [%s] %s", chanName, bot.levelId[chanId], bot.level[chanId], bot.userName[chanId], added, message)
		return result
	} else {
		if bot.level[chanId] == "" {
			return "Level not selected :<"
		} else {
		result := fmt.Sprintf("Last played level: %s by %s", bot.level[chanId], bot.userName[chanId])
		return result
		}
	}
	return "No idea what happened!?"
}

func (bot *Bot) getStats(channel string) string {
	
	chanId := bot.getChanId(channel)

	var allCount int
	var playCount int
	allLevels := db.QueryRow("SELECT count(Played) FROM Levels WHERE StreamID=?", chanId).Scan(&allCount)
	if allLevels != nil {
		log.Fatalf("Cannot count levels: %s", allLevels.Error())
	}
	playedLevels := db.QueryRow(" SELECT count(Played) FROM Levels WHERE StreamID=? AND Played=1", chanId).Scan(&playCount)
	if playedLevels != nil {
		log.Fatalf("Cannot count played levels: %s", playedLevels.Error())
	}
	result := fmt.Sprintf("Streamer has played %d levels from %d", playCount, allCount)
        return result
}

func (bot *Bot) getChanId(channel string) int {
        var chanId int
        for i := range bot.channel {
                if channel == bot.channel[i] {
                        chanId = i
                }
        }
        if chanId == 0 {
                log.Fatalf("chanId not found\n")
        }
        return chanId
}

