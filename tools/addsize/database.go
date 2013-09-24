package main

import (
	"crypto/md5"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

var db *DB

type Book struct {
	Id          string `bson:"_id"`
	Title       string
	Author      []string
	Contributor string
	Publisher   string
	Description string
	Subject     []string
	Date        string
	Lang        []string
	Isbn        string
	Type        string
	Format      string
	Source      string
	Relation    string
	Coverage    string
	Rights      string
	Meta        string
	File        bson.ObjectId
	FileSize    int
	Cover       bson.ObjectId
	CoverSmall  bson.ObjectId
	Active      bool
	Keywords    []string
}

type News struct {
	Date time.Time
	Text string
}

type DB struct {
	session *mgo.Session
	books   *mgo.Collection
	user    *mgo.Collection
	news    *mgo.Collection
	stats   *mgo.Collection
	mr      *MR
}

func initDB() *DB {
	var err error
	d := new(DB)
	d.session, err = mgo.Dial(DB_IP)
	if err != nil {
		panic(err)
	}

	database := d.session.DB(DB_NAME)
	d.books = database.C(BOOKS_COLL)
	d.user = database.C(USERS_COLL)
	d.news = database.C(NEWS_COLL)
	d.stats = database.C(STATS_COLL)
	d.mr = NewMR(database)
	return d
}

func (d *DB) Close() {
	d.session.Close()
}

func md5Pass(pass string) []byte {
	h := md5.New()
	hash := h.Sum(([]byte)(PASS_SALT + pass))
	return hash
}

func (d *DB) SetPassword(user string, pass string) error {
	hash := md5Pass(pass)
	return d.user.Update(bson.M{"user": user}, bson.M{"$set": bson.M{"pass": hash}})
}

func (d *DB) UserValid(user string, pass string) bool {
	hash := md5Pass(pass)
	n, err := d.user.Find(bson.M{"user": user, "pass": hash}).Count()
	if err != nil {
		return false
	}
	return n != 0
}

func (d *DB) UserRole(user string) string {
	type result struct {
		Role string
	}
	res := result{}
	err := d.user.Find(bson.M{"user": user}).One(&res)
	if err != nil {
		return ""
	}
	return res.Role
}

func (d *DB) AddNews(text string) error {
	var news News
	news.Text = text
	news.Date = time.Now()
	return d.news.Insert(news)
}

func (d *DB) GetNews(num int, days int) (news []News, err error) {
	query := bson.M{}
	if days != 0 {
		duration := time.Duration(-24*days) * time.Hour
		date := time.Now().Add(duration)
		query = bson.M{"date": bson.M{"$gt": date}}
	}
	q := d.news.Find(query).Sort("-date").Limit(num)
	err = q.All(&news)
	return
}

func (d *DB) InsertStats(stats interface{}) error {
	return d.stats.Insert(stats)
}

func (d *DB) InsertBook(book interface{}) error {
	return d.books.Insert(book)
}

func (d *DB) RemoveBook(id bson.ObjectId) error {
	return d.books.Remove(bson.M{"_id": id})
}

func (d *DB) UpdateBook(id bson.ObjectId, data interface{}) error {
	return d.books.Update(bson.M{"_id": id}, bson.M{"$set": data})
}

/* optional parameters: length and start index
 *
 * Returns: list of books, number found and err
 */
func (d *DB) GetBooks(query bson.M, r ...int) (books []Book, num int, err error) {
	var start, length int
	if len(r) > 0 {
		length = r[0]
		if len(r) > 1 {
			start = r[1]
		}
	}
	q := d.books.Find(query).Sort("-_id")
	num, err = q.Count()
	if err != nil {
		return
	}
	if start != 0 {
		q = q.Skip(start)
	}
	if length != 0 {
		q = q.Limit(length)
	}

	err = q.All(&books)
	for i, b := range books {
		books[i].Id = bson.ObjectId(b.Id).Hex()
	}
	return
}

/* Get the most visited books
 */
func (d *DB) GetVisitedBooks(num int) (books []Book, err error) {
	bookId, err := d.mr.GetMostVisited(num, d.stats)
	if err != nil {
		return nil, err
	}

	books = make([]Book, num)
	for i, id := range bookId {
		d.books.Find(bson.M{"_id": id}).One(&books[i])
		books[i].Id = bson.ObjectId(books[i].Id).Hex()
	}
	return
}

/* Get the most downloaded books
 */
func (d *DB) GetDownloadedBooks(num int) (books []Book, err error) {
	bookId, err := d.mr.GetMostDownloaded(num, d.stats)
	if err != nil {
		return nil, err
	}

	books = make([]Book, num)
	for i, id := range bookId {
		d.books.Find(bson.M{"_id": id}).One(&books[i])
		books[i].Id = bson.ObjectId(books[i].Id).Hex()
	}
	return
}

/* optional parameters: length and start index
 *
 * Returns: list of books, number found and err
 */
func (d *DB) GetNewBooks(r ...int) (books []Book, num int, err error) {
	return d.GetBooks(bson.M{"$nor": []bson.M{{"active": true}}}, r...)
}

func (d *DB) BookActive(id bson.ObjectId) bool {
	var book Book
	err := d.books.Find(bson.M{"_id": id}).One(&book)
	if err != nil {
		return false
	}
	return book.Active
}

func (d *DB) GetFS(prefix string) *mgo.GridFS {
	return d.session.DB(DB_NAME).GridFS(prefix)
}

func (d *DB) GetTags(numTags int) ([]string, error) {
	return d.mr.GetTags(numTags, d.books)
}

type Visits struct {
	Date  int64 "_id"
	Count int   "value"
}

func (d *DB) GetHourVisits(start time.Time) ([]Visits, error) {
	return d.mr.GetHourVisits(start, d.stats)
}

func (d *DB) GetDayVisits(start time.Time) ([]Visits, error) {
	return d.mr.GetDayVisits(start, d.stats)
}

func (d *DB) GetMonthVisits(start time.Time) ([]Visits, error) {
	return d.mr.GetMonthVisits(start, d.stats)
}