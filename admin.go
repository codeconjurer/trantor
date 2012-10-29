package main

import (
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"strings"
)

type settingsData struct {
	S Status
}

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	sess := GetSession(r)
	if sess.User == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method == "POST" {
		current_pass := r.FormValue("currpass")
		pass1 := r.FormValue("password1")
		pass2 := r.FormValue("password2")
		switch {
		case !db.UserValid(sess.User, current_pass):
			sess.Notify("Password error!", "The current password given don't match with the user password. Try again", "error")
		case pass1 != pass2:
			sess.Notify("Passwords don't match!", "The new password and the confirmation password don't match. Try again", "error")
		default:
			db.SetPassword(sess.User, pass1)
			sess.Notify("Password updated!", "Your new password is correctly set.", "success")
		}
	}

	var data settingsData
	data.S = GetStatus(w, r)
	loadTemplate(w, "settings", data)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	sess := GetSession(r)
	if sess.User == "" {
		http.NotFound(w, r)
		return
	}

	var titles []string
	var isNew bool
	ids := strings.Split(r.URL.Path[len("/delete/"):], "/")
	for _, idStr := range ids {
		if idStr == "" {
			continue
		}

		id := bson.ObjectIdHex(idStr)
		books, _, err := db.GetBooks(bson.M{"_id": id})
		if err != nil {
			sess.Notify("Book not found!", "The book with id '"+idStr+"' is not there", "error")
			continue
		}
		book := books[0]
		DeleteBook(book)
		db.RemoveBook(id)

		if !book.Active {
			isNew = true
		}
		titles = append(titles, book.Title)
	}
	if titles != nil {
		sess.Notify("Removed books!", "The books "+strings.Join(titles, ", ")+" are completly removed", "success")
	}
	sess.Save(w, r)
	if isNew {
		http.Redirect(w, r, "/new/", 307)
	} else {
		http.Redirect(w, r, "/", 307)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	sess := GetSession(r)
	if sess.User == "" {
		http.NotFound(w, r)
		return
	}
	id := bson.ObjectIdHex(r.URL.Path[len("/edit/"):])
	books, _, err := db.GetBooks(bson.M{"_id": id})
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var data bookData
	data.Book = books[0]
	data.S = GetStatus(w, r)
	loadTemplate(w, "edit", data)
}

func cleanEmptyStr(s []string) []string {
	var res []string
	for _, v := range s {
		if v != "" {
			res = append(res, v)
		}
	}
	return res
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	sess := GetSession(r)
	if sess.User == "" {
		http.NotFound(w, r)
		return
	}

	idStr := r.URL.Path[len("/save/"):]
	id := bson.ObjectIdHex(idStr)
	title := r.FormValue("title")
	publisher := r.FormValue("publisher")
	date := r.FormValue("date")
	description := r.FormValue("description")
	author := cleanEmptyStr(r.Form["author"])
	subject := cleanEmptyStr(r.Form["subject"])
	lang := cleanEmptyStr(r.Form["lang"])
	book := map[string]interface{}{"title": title,
		"publisher":   publisher,
		"date":        date,
		"description": description,
		"author":      author,
		"subject":     subject,
		"lang":        lang}
	book["keywords"] = keywords(book)
	err := db.UpdateBook(id, book)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sess.Notify("Book Modified!", "", "success")
	sess.Save(w, r)
	if db.BookActive(id) {
		http.Redirect(w, r, "/book/"+idStr, 307)
	} else {
		http.Redirect(w, r, "/new/", 307)
	}
}

type newBook struct {
	TitleFound  int
	AuthorFound int
	B           Book
}
type newData struct {
	S     Status
	Found int
	Books []newBook
}

func newHandler(w http.ResponseWriter, r *http.Request) {
	sess := GetSession(r)
	if sess.User == "" {
		http.NotFound(w, r)
		return
	}

	if len(r.URL.Path) > len("/new/") {
		http.ServeFile(w, r, NEW_PATH+r.URL.Path[len("/new/"):])
		return
	}

	res, num, _ := db.GetNewBooks()
	var data newData
	data.S = GetStatus(w, r)
	data.Found = num
	data.Books = make([]newBook, num)
	for i, b := range res {
		data.Books[i].B = b
		_, data.Books[i].TitleFound, _ = db.GetBooks(buildQuery("title:"+b.Title), 1)
		_, data.Books[i].AuthorFound, _ = db.GetBooks(buildQuery("author:"+strings.Join(b.Author, " author:")), 1)
	}
	loadTemplate(w, "new", data)
}

func storeHandler(w http.ResponseWriter, r *http.Request) {
	sess := GetSession(r)
	if sess.User == "" {
		http.NotFound(w, r)
		return
	}

	var titles []string
	ids := strings.Split(r.URL.Path[len("/store/"):], "/")
	for _, idStr := range ids {
		if idStr == "" {
			continue
		}

		id := bson.ObjectIdHex(idStr)
		books, _, err := db.GetBooks(bson.M{"_id": id})
		if err != nil {
			sess.Notify("Book not found!", "The book with id '"+idStr+"' is not there", "error")
			continue
		}
		book := books[0]
		path, err := StoreBook(book)
		if err != nil {
			sess.Notify("An error ocurred!", err.Error(), "error")
			log.Println("Error storing book '", book.Title, "': ", err.Error())
			continue
		}
		db.UpdateBook(id, bson.M{"active": true, "path": path})
		titles = append(titles, book.Title)
	}
	if titles != nil {
		sess.Notify("Store books!", "The books '"+strings.Join(titles, ", ")+"' are stored for public download", "success")
	}
	sess.Save(w, r)
	http.Redirect(w, r, "/new/", 307)
}
