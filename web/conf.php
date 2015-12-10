<?php
  # PDO MySQL connect string 
  # databasename - Name of mysql database
  # dbusername   - MySQL database username
  # dbpassword   - MySQL database password
  # host         - Hostname/IP of MySQL database (localhost by default)
  $conn = new PDO ('mysql:host=localhost;dbname=databasename;charset=utf8', 'dbusername', 'dbpassword');
?>
