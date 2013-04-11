package main

import (
	"git.gitorious.org/go-pkg/epubgo.git"
	"io"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

func ParseFile(id bson.ObjectId) (string, error) {
	book := map[string]interface{}{}

	e, err := OpenBook(id)
	if err != nil {
		return "", err
	}
	defer e.Close()

	for _, m := range e.MetadataFields() {
		data, err := e.Metadata(m)
		if err != nil {
			continue
		}
		switch m {
		case "creator":
			book["author"] = parseAuthr(data)
		case "description":
			book[m] = parseDescription(data)
		case "subject":
			book[m] = parseSubject(data)
		case "date":
			book[m] = parseDate(data)
		case "language":
			book["lang"] = data
		case "title", "contributor", "publisher":
			book[m] = cleanStr(strings.Join(data, ", "))
		case "identifier":
			attr, _ := e.MetadataAttr(m)
			for i, d := range data {
				if attr[i]["scheme"] == "ISBN" {
					book["isbn"] = d
				}
			}
		default:
			book[m] = strings.Join(data, ", ")
		}
	}
	title, _ := book["title"].(string)
	book["file"] = id
	cover, coverSmall := GetCover(e, title)
	book["cover"] = cover
	book["coversmall"] = coverSmall
	book["keywords"] = keywords(book)

	db.InsertBook(book)
	return title, nil
}

func OpenBook(id bson.ObjectId) (*epubgo.Epub, error) {
	fs := db.GetFS(FS_BOOKS)
	var reader readerGrid
	var err error
	reader.f, err = fs.OpenId(id)
	if err != nil {
		return nil, err
	}
	defer reader.f.Close()
	return epubgo.Load(reader, reader.f.Size())
}

type readerGrid struct {
	f *mgo.GridFile
}

func (r readerGrid) ReadAt(p []byte, off int64) (n int, err error) {
	_, err = r.f.Seek(off, 0)
	if err != nil {
		return
	}

	return r.f.Read(p)
}

func StoreNewFile(name string, file io.Reader) (bson.ObjectId, error) {
	fs := db.GetFS(FS_BOOKS)
	fw, err := fs.Create(name)
	if err != nil {
		return "", err
	}
	defer fw.Close()

	_, err = io.Copy(fw, file)
	id, _ := fw.Id().(bson.ObjectId)
	return id, err
}

func DeleteFile(id bson.ObjectId) error {
	fs := db.GetFS(FS_BOOKS)
	return fs.RemoveId(id)
}

func DeleteBook(book Book) {
	if book.Cover != "" {
		os.RemoveAll(book.Cover[1:])
	}
	if book.CoverSmall != "" {
		os.RemoveAll(book.CoverSmall[1:])
	}
	DeleteFile(book.File)
}

func validFileName(path string, title string, extension string) string {
	title = strings.Replace(title, "/", "_", -1)
	title = strings.Replace(title, "?", "_", -1)
	title = strings.Replace(title, "#", "_", -1)
	r, _ := utf8.DecodeRuneInString(title)
	folder := string(r)
	file := folder + "/" + title + extension
	_, err := os.Stat(path + file)
	for i := 0; err == nil; i++ {
		file = folder + "/" + title + "_" + strconv.Itoa(i) + extension
		_, err = os.Stat(path + file)
	}
	return file
}

func cleanStr(str string) string {
	str = strings.Replace(str, "&#39;", "'", -1)
	exp, _ := regexp.Compile("&[^;]*;")
	str = exp.ReplaceAllString(str, "")
	exp, _ = regexp.Compile("[ ,]*$")
	str = exp.ReplaceAllString(str, "")
	return str
}

func parseAuthr(creator []string) []string {
	exp1, _ := regexp.Compile("^(.*\\( *([^\\)]*) *\\))*$")
	exp2, _ := regexp.Compile("^[^:]*: *(.*)$")
	res := make([]string, len(creator))
	for i, s := range creator {
		auth := exp1.FindStringSubmatch(s)
		if auth != nil {
			res[i] = cleanStr(strings.Join(auth[2:], ", "))
		} else {
			auth := exp2.FindStringSubmatch(s)
			if auth != nil {
				res[i] = cleanStr(auth[1])
			} else {
				res[i] = cleanStr(s)
			}
		}
	}
	return res
}

func parseDescription(description []string) string {
	str := cleanStr(strings.Join(description, ", "))
	exp, _ := regexp.Compile("<[^>]*>")
	str = exp.ReplaceAllString(str, "")
	str = strings.Replace(str, "&amp;", "&", -1)
	str = strings.Replace(str, "&lt;", "<", -1)
	str = strings.Replace(str, "&gt;", ">", -1)
	str = strings.Replace(str, "\\n", "\n", -1)
	return str
}

func parseSubject(subject []string) []string {
	var res []string
	for _, s := range subject {
		res = append(res, strings.Split(s, " / ")...)
	}
	return res
}

func parseDate(date []string) string {
	if len(date) == 0 {
		return ""
	}
	return strings.Replace(date[0], "Unspecified: ", "", -1)
}

func keywords(b map[string]interface{}) (k []string) {
	title, _ := b["title"].(string)
	k = strings.Split(title, " ")
	author, _ := b["author"].([]string)
	for _, a := range author {
		k = append(k, strings.Split(a, " ")...)
	}
	publisher, _ := b["publisher"].(string)
	k = append(k, strings.Split(publisher, " ")...)
	subject, _ := b["subject"].([]string)
	k = append(k, subject...)
	return
}
