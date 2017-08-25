Bash Telegram Bot
=================

# Run
    go run main.go

# Configurations
Bot configuration must have name config.yml.

    telegram:
      token : "telegram_token"
      cert : "path/to/certificate"
      pkey : "path/to/public_key"
      host : ""
      port : ""
      pool_size : 
      timeout : 
      debug : 
      
If field cert or pkey left empty, then bot will get updates by getUpdate method. Otherwise, webhooks will be used.
      
Database configuration must have name db.yml.

    database:
      host : "localhost"
      port : "3301"
      user : ""
      pass : ""
      timeout:       1
      reconnect:     1
      max_reconnects: 5

# Usage
Bot can send random quotes from the bashorg. User can like or dislike quotes. Liked quotes will be saved. The user will be able to see them and delete them, if he wants.
