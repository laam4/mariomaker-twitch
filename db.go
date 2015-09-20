// db
package main

import (
	"log"
	"time"
	"fmt"
//	"io/ioutil"
//	"os"
	"strings"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

func (bot *Bot) getLevel(mod bool, channel string) string {
	if mod {
        	db, err := sql.Open("mysql", "mario:salakala@unix(/var/run/mysqld/mysqld.sock)/mariomaker")
        	checkErr(err)

	        var levelid int
       		var sid int
        	var message string
        	var played bool
        	var added string
		var passed string
        	var sid2 int
        	var name string
        	channelname := strings.Replace(channel, "#", "", 1)
		levels := db.QueryRow("SELECT * FROM Levels LEFT JOIN Streamers ON Streamers.StreamID = Levels.StreamID WHERE Played=0 AND Name=? ORDER BY RAND() LIMIT 1", channelname).Scan(&levelid, &sid, &bot.creator, 
&bot.level, &message, &played, &added, &passed, &sid2, &name)
	        switch {
        	case levels == sql.ErrNoRows:
                	//fmt.Printf("No levels\n")
			return "No levels :<<"
        	case levels != nil:
                	log.Fatal(levels)
        	default:
                	fmt.Printf("New random level chosen, maybe? #%d %s by %s\n", levelid, bot.level, bot.creator)
        	}

                update, err := db.Prepare("UPDATE Levels SET Played=1,Passed=? WHERE LevelID=?")
		now := time.Now().Format(time.RFC3339)
                insert, err := update.Exec(now, levelid)
                affect, err := insert.RowsAffected()
                checkErr(err)
                fmt.Println(affect)

		//currentlevel := level
		result := fmt.Sprintf("@%s: %s by %s | [%s] %s", name, bot.level, bot.creator, added, message)
		return result
		defer db.Close()
	} else {
		if bot.level == "" {
			fmt.Printf("level %s creator %s", bot.level, bot.creator)
			return "Level not chosen :<"
		} else {
		result := fmt.Sprintf("Last random level: %s by %s", bot.level, bot.creator)
		return result
		}
	}
	return "Something went wrong, oops."
}

func (bot *Bot) writeLevelDB(channel string, username string, usermessage string, levelid string) {
	db, err := sql.Open("mysql", "mario:salakala@unix(/var/run/mysqld/mysqld.sock)/mariomaker")
        checkErr(err)

	var Streamer int
	channelname := strings.Replace(channel, "#", "", 1)
	row := db.QueryRow("SELECT StreamID FROM Streamers WHERE Name=?", channelname).Scan(&Streamer)
	switch {
	case row == sql.ErrNoRows:
		fmt.Printf("No streamer ID\n")
	case row != nil:
		fmt.Printf("Error")
	default:
		fmt.Printf("StreamerId for %s is %d\n", channel, Streamer)
	}
	
	var Duplicate string
	row2 := db.QueryRow("SELECT Level FROM Levels WHERE Level=? AND StreamID=?", levelid,Streamer).Scan(&Duplicate)
        switch {
        case row2 == sql.ErrNoRows:
                fmt.Printf("No such level, Adding...\n")
                level, err := db.Prepare("INSERT Levels SET StreamID=?,Nick=?,Level=?,Message=?,Added=?")
                now := time.Now().Format(time.RFC3339)
                insert, err := level.Exec(Streamer, username, levelid, usermessage, now)
                affect, err := insert.RowsAffected()
                checkErr(err)
                fmt.Println(affect)
        case row2 != nil:
                log.Fatal(err)
		//fmt.Printf("Another duplicate?\n")
        default:
                fmt.Printf("Duplicate level, not adding...\n")
        }

//if strings.Compare(levelid,Duplicate) != 0 {
	defer db.Close()
}

func (bot *Bot) InitDB() {
        db, err := sql.Open("mysql", "mario:salakala@unix(/var/run/mysqld/mysqld.sock)/mariomaker")
        checkErr(err)

	db.Exec("CREATE TABLE IF NOT EXISTS Streamers (      StreamID MEDIUMINT NOT NULL AUTO_INCREMENT,      Name VARCHAR(25) NOT NULL UNIQUE,      PRIMARY KEY (StreamID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
	db.Exec("CREATE TABLE IF NOT EXISTS Levels ( LevelID MEDIUMINT NOT NULL AUTO_INCREMENT, StreamID MEDIUMINT NOT NULL, Nick VARCHAR(25) NOT NULL, Level VARCHAR(22) NOT NULL, Message VARCHAR(255) NOT NULL, Played BOOLEAN NOT NULL, Added DATETIME NOT NULL, Passed DATETIME NOT NULL,PRIMARY KEY (LevelID) ) ENGINE=MyISAM DEFAULT CHARSET=utf8;")
	checkErr(err)
        var Streamer int
        for i := range bot.channel {
		channelname := strings.Replace(bot.channel[i], "#", "", 1)
		row := db.QueryRow("SELECT StreamID FROM Streamers WHERE Name=?", channelname).Scan(&Streamer)
	        switch {
		case row == sql.ErrNoRows:
                	fmt.Printf("No streamer ID\n")
                	caster, err := db.Prepare("INSERT Streamers SET Name=?")
                	addstream, err := caster.Exec(channelname)
	                last, err := addstream.LastInsertId()
           		checkErr(err)
                	fmt.Println(last)
                	Streamer := last
                	fmt.Printf("New streamId for %s is %d\n", bot.channel[i], Streamer)
        	case row != nil:
                	fmt.Printf("Error")
        	default:
                	fmt.Printf("StreamerId for %s is %d\n", bot.channel[i], Streamer)
		}
        }
	defer db.Close()
	return
}

func (bot *Bot) getStats(channel string) string {
        db, err := sql.Open("mysql", "mario:salakala@unix(/var/run/mysqld/mysqld.sock)/mariomaker")
        checkErr(err)

        var Streamer int
        channelname := strings.Replace(channel, "#", "", 1)
        row := db.QueryRow("SELECT StreamID FROM Streamers WHERE Name=?", channelname).Scan(&Streamer)
        switch {
        case row == sql.ErrNoRows:
                fmt.Printf("No streamer ID\n")
        case row != nil:
                fmt.Printf("Error")
        default:
                fmt.Printf("StreamerId for %s is %d\n", channel, Streamer)
        }

	var allCount int
	var playCount int
	all := db.QueryRow("SELECT count(Played) FROM Levels WHERE StreamID=?", Streamer).Scan(&allCount)
	play := db.QueryRow(" SELECT count(Played) FROM Levels WHERE StreamID=? AND Played=1", Streamer).Scan(&playCount)
	fmt.Printf("ALL: %d Played: %d\n", all, play)
	result := fmt.Sprintf("%s has %d levels in database, played %d", channel, allCount, playCount)
	defer db.Close()
        return result
}


func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}
