package main

import (
	"code.google.com/p/gorilla/securecookie"
	"code.google.com/p/gorilla/sessions"
	"net/http"
)

var sesStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(64))

type Notification struct {
	Title string
	Msg   string
	Type  string /* error, info or success */
}

type Session struct {
	User  string
	Notif []Notification
	S     *sessions.Session
}

func getNotif(session *sessions.Session) []Notification {
	msgs := session.Flashes("nMsg")
	titles := session.Flashes("nTitle")
	tpes := session.Flashes("nType")
	notif := make([]Notification, len(msgs))
	for i, m := range msgs {
		msg, _ := m.(string)
		title, _ := titles[i].(string)
		tpe, _ := tpes[i].(string)
		notif[i] = Notification{title, msg, tpe}
	}
	return notif
}

func GetSession(r *http.Request) (s *Session) {
	s = new(Session)
	var err error
	s.S, err = sesStore.Get(r, "session")
	if err == nil && !s.S.IsNew {
		s.User, _ = s.S.Values["user"].(string)
		s.Notif = getNotif(s.S)
	}
	return
}

func (s *Session) LogIn(user string) {
	s.User = user
	s.S.Values["user"] = user
}

func (s *Session) LogOut() {
	s.S.Values["user"] = ""
}

func (s *Session) Notify(title, msg, tpe string) {
	s.S.AddFlash(msg, "nMsg")
	s.S.AddFlash(title, "nTitle")
	s.S.AddFlash(tpe, "nType")
}

func (s *Session) Save(w http.ResponseWriter, r *http.Request) {
	sesStore.Save(r, w, s.S)
}
