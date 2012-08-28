package main

const (
	DB_IP             = "127.0.0.1"
	DB_NAME           = "trantor"
	BOOKS_COLL        = "books"
	NEW_BOOKS_COLL    = "new"
	USERS_COLL        = "users"
	PASS_SALT         = "ImperialLibSalt"
	TAGS_DISPLAY      = 50
	SEARCH_ITEMS_PAGE = 10
	TEMPLATE_PATH     = "templates/"
	BOOKS_PATH        = "books/"
	COVER_PATH        = "cover/"
	NEW_PATH          = "new/"
	RESIZE_CMD        = "/usr/bin/convert -resize 300 -quality 60 "
	RESIZE_THUMB_CMD  = "/usr/bin/convert -resize 60 -quality 60 "
)