<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Super Mario Maker Levels</title>
<link href="mario.css" rel="stylesheet" type="text/css" />
<link rel="shortcut icon" href="favicon.ico" type="image/x-icon">
<link rel="icon" href="favicon.ico" type="image/x-icon">
</head>
<img class="center" src="mario.png" alt="Mario" style="width:714px;height:559px;">
<?php
require 'conf.php';
$sql = "SELECT Name,StreamID FROM Streamers";

$getlist = $conn->prepare($sql);
$getlist->execute();
$res = $getlist->fetchAll();

echo   '<form method="get" action="level.php">
        <select name="i">
        <option>Select Streamer</option>';   
		
	foreach ($res as $red){
        	echo '<option name="dro" value=' . $red['StreamID'] . '>' .$red['Name']  . '</option>';
	}

echo   '</select>
        <input type="submit" value="Let\'s a go!">
        </form>';
        ?>	
</body>
</html>



