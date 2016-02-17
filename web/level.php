<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Super Mario Maker Levels</title>
<link href="mario.css" rel="stylesheet" type="text/css" />
</head>
<p><a href="index.php">Back to main menu</a></p>
<?php
require 'conf.php';
	function makeHeaderLink($value, $key, $col, $dir) {
		$out = "<a href=\"" . $_SERVER['SCRIPT_NAME'] . "?i=" . $_GET['i'] . "&c=";		
		//set column query string value
		switch($key) {
			case "LevelID":
				$out .= "1";
				break;
			case "Nick":
				$out .= "2";
				break;
			case "Level":
				$out .= "3";
				break;
			case "Played":
				$out .= "4";
				break;
			case "Skipped":
				$out .= "5";
				break;
                        case "Passed":
                                $out .= "6";
                                break;
                        case "Added":
                                $out .= "7";
                                break;
                        case "Message":
                                $out .= "8";
                                break;
			case "Comment":
				$out .= "9";
				break;
			default:
				$out .= "0";
			}
			
			$out .= "&d=";
		
		//reverse sort if the current column is clicked
		if($key == $col) {
			switch($dir) {
				case "ASC":
					$out .= "1";
					break;
				default:
					$out .= "0";
			}
		}
		else {
			//pass on current sort direction
			switch($dir) {
				case "ASC":
					$out .= "0";
					break;
				default:
					$out .= "1";
			}
		}
			
		//complete link
		$out .= "\">$value</a>";
			
		return $out;
	}
	switch($_GET['c']) {
		case "1":
			$col = "LevelID";
			break;
		case "2":
			$col = "Nick";
			break;
		case "3":
			$col = "Level";
			break;
		case "4":
			$col = "Played";
			break;
                case "5":
                        $col = "Skipped";
                        break;
                case "6":
                        $col = "Passed";
                        break;
                case "7":
                        $col = "Added";
                        break;
		case "8":
			$col = "Message";
			break;
		case "9":
			$col = "Comment";
			break;
		default:
			$col = "LevelID";
	}
	if($_GET['d'] == "1") {
		$dir = "DESC";
	}
	else {
		$dir = "ASC";
	}
	
	if(isset($_GET['i'])) {
                @$id=$_GET['i'];
                if(strlen($id) > 0 and !is_numeric($id)) {
                echo "ERROR: Not valid value for streamer!";
                exit;
		}
	} 
	else {
                echo "ERROR: Streamer not selected!";
                exit;
	}	

	$conn->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
	$conn->setAttribute(PDO::ATTR_EMULATE_PREPARES, false);
	$que = "SELECT LevelID, Nick, Level, Message, Comment, Played, Skipped, Added, Passed FROM Levels WHERE StreamID= :id ORDER BY $col $dir;";
	$sth = $conn->prepare($que);
	$sth->bindValue(':id', $id, PDO::PARAM_INT);
	$sth->execute();
	$rows = 0;
	$played = 0;
	$skipped = 0;
        echo "<table class=\"bordered\" cellspacing=\"0\">\n";
        echo "<tr>";
		echo "<th>" . makeHeaderLink("ID", "LevelID", $col, $dir) . "</th>";
                echo "<th>" . makeHeaderLink("Name", "Nick", $col, $dir) . "</th>";
                echo "<th>" . makeHeaderLink("Level", "Level", $col, $dir) . "</th>";
                echo "<th>" . makeHeaderLink("Played", "Played", $col, $dir) . "</th>";
		echo "<th>" . makeHeaderLink("Skipped", "Skipped", $col, $dir) . "</th>";
                echo "<th>" . makeHeaderLink("Selected", "Passed", $col, $dir) . "</th>";
                echo "<th>" . makeHeaderLink("Added", "Added", $col, $dir) . "</th>";
                echo "<th>" . makeHeaderLink("Message", "Message", $col, $dir) . "</th>";
		echo "<th>" . makeHeaderLink("Comment", "Comment", $col, $dir) . "</th>";
                echo "</tr>\n";
                while($row = $sth->fetch(PDO::FETCH_ASSOC)) {
                    echo "<tr><td>$row[LevelID]</td><td nowrap>$row[Nick]</td><td nowrap>$row[Level]</td><td style=\"text-align:center\"><img src=\"" . $row['Played'] . ".png\" alt=\"img\" height=\"16\" width=\"16\"></td><td style=\"text-align:center\"><img src=\"" . $row['Skipped'] . ".png\" alt=\"img\" height=\"16\" width=\"16\"></td><td nowrap>$row[Passed]</td><td nowrap>$row[Added]</td><td>$row[Message]</td><td>$row[Comment]</td></tr>\n";
		    if ($row['Played']) {
			if ($row['Skipped']) {
				$skipped++;
			} 
			else {
				$played++;
			}
		    }
		    $rows++;
                }
                echo "</table>\n";
		echo "<span style=\"background-color:white\">Total: $played levels played and $skipped levels skipped out of $rows levels</span>";
        ?> </body>
</html>
