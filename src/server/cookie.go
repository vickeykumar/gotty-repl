package server

import (
	"encoding/base64"
	"github.com/gorilla/websocket"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
	"strconv"
	"webtty"
	"utils"
	"user"
	"errors"
)

const MAX_CONN_PER_BROWSER = 1

// handling for cookies and incrementing session cookie counter
func IncrementCounterCookies(rw http.ResponseWriter, req *http.Request) {
	cookie, err := req.Cookie("Session-Counter")
	if err == http.ErrNoCookie {
		cookie = &http.Cookie{
			Name:  "Session-Counter",
			Value: "0",
		}
	} else if err != nil {
		log.Println("unknown error while getting cookie: ", err.Error())
		return
	}
	sessionCount, _ := strconv.Atoi(cookie.Value)
	sessionCount++
	cookie.Value = strconv.Itoa(sessionCount)
	http.SetCookie(rw, cookie)
	log.Println("cookie set: ", cookie)
}

func DecrementCounterCookies(rw http.ResponseWriter, req *http.Request) {
	// decrementing cookie
	cookie, err := req.Cookie("Session-Counter")
	if err == http.ErrNoCookie {
		cookie = &http.Cookie{
			Name:  "Session-Counter",
			Value: "0",
		}
	} else {
		log.Println("unknown error while getting cookie: ", err.Error())
		return
	}
	sessionCount, _ := strconv.Atoi(cookie.Value)
	if sessionCount > 0 {
		sessionCount--
		cookie.Value = strconv.Itoa(sessionCount)
		http.SetCookie(rw, cookie)
		log.Println("cookie dec: ", cookie)
	}
}

func GetCounterCookieValue(rw http.ResponseWriter, req *http.Request) int {
	cookie, err := req.Cookie("Session-Counter")
	if err != nil {
		log.Println("Error while getting cookie: ", err.Error())
		return 0
	}
	sessionCount, _ := strconv.Atoi(cookie.Value)
	return sessionCount
}

func WriteMessageToTerminal(conn *websocket.Conn, message string) {
	safeMessage := base64.StdEncoding.EncodeToString([]byte(message))
	err := conn.WriteMessage(websocket.TextMessage, []byte(append([]byte{webtty.Output}, []byte(safeMessage)...)))
	if err != nil {
		log.Println("err while writing: ", err)
	}
}


var session_store *sessions.CookieStore

func Init_SessionStore(secret string) {
	session_store = sessions.NewCookieStore([]byte(secret))
}

func Get_SessionStore() *sessions.CookieStore {
	return session_store
}

func init() {
    session_db_handle := user.GetUserDBHandle()
    if session_db_handle == nil {
    	panic(errors.New("uninitialized session_db_handle!!"))
    }
    secret , err := session_db_handle.Fetch([]byte(user.SESSION_KEY))
    log.Println("secret: ", secret, err)
    if err != nil {
    	panic(errors.New("No secret found for session_cookie."))
    }
    // init one time session store using SESSION_KEY
    Init_SessionStore(string(secret))

}


func Set_SessionCookie(rw http.ResponseWriter, req *http.Request, session user.UserSession) (err error) {
		session_cookie, err := session_store.Get(req, "user-session")
		if err != nil {
			// log and move on, u can still save
			log.Println("Error: while getting cookie err: ", err.Error())
		}
		// Set some session values.
		session_cookie.Values["uid"] = session.Uid
		session_cookie.Values["sessionID"] = session.SessionID
		session_cookie.Values ["loggedIn"] = session.LoggedIn
		session_cookie.Values ["expirationTime"] = session.ExpirationTime

		// set maxage of the session
		session_cookie.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   int(session.ExpirationTime-utils.GetUnixMilli())/1000,
		}

		log.Println("session cookie save: ", int(session.ExpirationTime-utils.GetUnixMilli())/1000, session_cookie)
		return session_store.Save(req, rw, session_cookie)
}


func Delete_SessionCookie(rw http.ResponseWriter, req *http.Request, session user.UserSession) (err error) {
		session_cookie, err := session_store.Get(req, "user-session")
		if err != nil {
			log.Println("Error: while getting cookie err: ", err.Error())
		}
		// Set some session values.
		session_cookie.Values["uid"] = session.Uid
		session_cookie.Values["sessionID"] = session.SessionID
		session_cookie.Values ["loggedIn"] = false

		// set maxage of the session
		session_cookie.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   -1,
		}

		return session_store.Save(req, rw, session_cookie)
}


func Is_UserLoggedIn(rw http.ResponseWriter, req *http.Request) (loggedin bool) {
	session_cookie, err := session_store.Get(req, "user-session")
	val := session_cookie.Values["loggedIn"]
	loggedin, ok := val.(bool);
	log.Println ("loggedIn: ", loggedin, err)
	if !ok {
		return false
	}
	return loggedin
}

func Get_Uid(rw http.ResponseWriter, req *http.Request) (uid string) {
	session_cookie, _ := session_store.Get(req, "user-session")
	val := session_cookie.Values["uid"]
	uid, ok := val.(string);
	if !ok {
		return ""
	}
	return uid
}

func Get_SessionID(rw http.ResponseWriter, req *http.Request) (sessionid string) {
	session_cookie, _ := session_store.Get(req, "user-session")
	val := session_cookie.Values["sessionID"]
	sessionid, ok := val.(string);
	if !ok {
		return ""
	}
	return sessionid
}

func Get_ExpirationTime(rw http.ResponseWriter, req *http.Request) (e int64) {
	session_cookie, _ := session_store.Get(req, "user-session")
	val := session_cookie.Values["expirationTime"]
	e, ok := val.(int64);
	if !ok {
		return utils.GetUnixMilli()
	}
	return e
}

func Get_SessionCookie(rw http.ResponseWriter, req *http.Request) (session user.UserSession) {
		session.Uid = Get_Uid(rw, req)
		session.SessionID = Get_SessionID(rw, req)
		session.LoggedIn = Is_UserLoggedIn(rw, req)
		session.ExpirationTime = Get_ExpirationTime(rw, req)
		return session
}

