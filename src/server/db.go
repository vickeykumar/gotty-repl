package server

import (
	"encoding/json"
	"fmt"
	"github.com/nobonobo/unqlitego"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"io/ioutil"
	"utils"
	"html/template"
	"bytes"
	"user"
)

const FEEDBACK_DB = utils.GOTTY_PATH + "/feedback.db"

var feedback_db_handle *unqlitego.Database

type feedback struct {
	Name    string
	Email   string
	Message string
}

func InitFeedbackDBHandle() {
	var err error
	feedback_db_handle, err = unqlitego.NewDatabase(FEEDBACK_DB)
	if err != nil {
		log.Println("ERROR: Error while creating feedback DB handle : ", err.Error())
		os.Exit(3)
	}
	log.Println("Successfully initialized fb handle: ", feedback_db_handle)
}

func CloseFeedbackDBHandle() {
	err := feedback_db_handle.Close()
	if err != nil {
		log.Println("ERROR: Error while closing feedback DB handle : ", err.Error())
	}
	feedback_db_handle = nil
}

func StoreFeedbackData(fb *feedback) error {
	data, err := json.Marshal(*fb)
	if err != nil {
		log.Println("ERROR: Error while marshalling feedback data. Error: ", err.Error())
		return err
	}
	log.Println("got feedback data: ", fb)
	timestamp := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	err = feedback_db_handle.Store([]byte(timestamp), data)
	if err != nil {
		log.Println("ERROR: Failed to store the feedbackdata, error: ", err.Error())
		return err
	}
	err = feedback_db_handle.Commit()
	if err != nil {
		log.Println("ERROR: Failed to commit the feedbackdata to disk, error: ", err.Error())
	}
	return err
}

func handleFeedback(rw http.ResponseWriter, req *http.Request) {
	log.Println("method:", req.Method)
	if req.Method == "POST" {
		req.ParseForm()
		var fb feedback
		fb.Name = strings.Join(req.Form["name"], "")
		fb.Email = strings.Join(req.Form["email"], "")
		fb.Message = strings.Join(req.Form["message"], "")

		err := StoreFeedbackData(&fb)
		if err == nil {
			fmt.Fprintf(rw, "Thanks for your Feedback !!")
		} else {
			fmt.Fprintf(rw, "Server Error: Unsupported Operation !!")
		}
	}
}


func handleLoginSession(rw http.ResponseWriter, req *http.Request) {
	log.Println("method:", req.Method)
	if req.Method == "POST" {
		req.ParseForm()
		//var session UserSession
		for key, val := range req.Form {
			log.Println("%s: %s", key, val);
		}
		body, err := ioutil.ReadAll(req.Body)
	    if err != nil {
	        panic(err)
	    }
    	//log.Println("body: ", string(body))
		var session user.UserSession
	    err = json.Unmarshal(body, &session)
	    if err != nil {
	        panic(err)
	    }

	    log.Println("before user: ", *session.User)
	    // fill other fields from lower hierarchy
	    session.Update(session.User)
	    session.LogIn()

	    log.Println("session: ", session)

	    err = user.UpdateAndStoreSessionData(session.Uid, session.SessionID, &session, false)
		if err == nil {
			log.Println("Session data Successfully written to the SESSION_DB.")
			// now can write/update session cookie here.
		} else {
			log.Println("ERROR: Failed to store the session data in SESSION_DB, error: ", err.Error())
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = Set_SessionCookie(rw, req, session)
		if err != nil {
			log.Println("Error: Set_SessionCookie Failed: ", err.Error())
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if req.Method == "GET" {
		log.Println("handleLoginSession: GET");
		req.ParseForm()
		//var session UserSession
		for key, val := range req.Form {
			log.Println("%s: %s", key, val);
		}

		var session user.UserSession
		session = Get_SessionCookie(rw, req)
		// verify the session in local db
		if (user.IsSessionExpired(session.Uid, session.SessionID)==true) {
			session.LogOut()
		}
		log.Println("session returned: ",session)
		rw.Write(utils.JsonMarshal(session))
	}
}


func handleLogoutSession(rw http.ResponseWriter, req *http.Request) {
	log.Println("method:", req.Method)
	if req.Method == "POST" {
		req.ParseForm()
		//var session UserSession
		for key, val := range req.Form {
			log.Println("%s: %s", key, val);
		}
		body, err := ioutil.ReadAll(req.Body)
		    if err != nil {
		        panic(err)
		    }
	    	log.Println("body: ", string(body))
		var session user.UserSession
		session = Get_SessionCookie(rw, req )
		log.Println("deleting user: "+session.Uid+ " with sessionID: "+session.SessionID+" from SESSION_COOKIE STORE")
		err = Delete_SessionCookie(rw, req, session)
		if err != nil {
			log.Println("ERROR: deleting user: "+session.Uid+" from SESSION_COOKIE STORE: "+err.Error())
		}
		err = user.UpdateAndStoreSessionData(session.Uid, session.SessionID, &session, true)	// store after deleting this session id(true)
		if err == nil {
			log.Println("Session data Successfully written to the SESSION_DB.")
			// now can write/update session cookie here.
		} else {
			log.Println("ERROR: Failed to store the session data in SESSION_DB, error: ", err.Error())
			http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	} else if req.Method == "GET" {
		log.Println("Error: invalid request type: GET")
		http.Error(rw, "Internal Server Error: invalid request type", http.StatusInternalServerError)
		return
	}
}

func handleUserProfileJson(rw http.ResponseWriter, req *http.Request, status int, up user.UserProfile) {
	rw.WriteHeader(status)
	// nullify the session map before sending probably we will not need it.
	up.SessionMap = nil
	rw.Write(utils.JsonMarshal(up))
}

func handleUserProfile(rw http.ResponseWriter, req *http.Request) {
	log.Println("handleUserProfile: method:", req.Method)
	if req.Method == "GET" {
		req.ParseForm()
		for key, val := range req.Form {
			log.Println("%s: %s", key, val);
		}

		var json bool

		query, ok := req.Form["q"]
		if !ok || len(query) == 0 {
			json=false
		} else {
			if query[0] == "json" {
				// json data is asked
				json = true
			}
		}
		

		var up user.UserProfile
		var session user.UserSession
		session = Get_SessionCookie(rw, req)
		// verify the session in local db
		if (user.IsSessionExpired(session.Uid, session.SessionID)==true) {
			session.LogOut()
			//return
			if json == true {
				handleUserProfileJson(rw, req, http.StatusUnauthorized, up)
				return
			}
			errorHandler(rw, req, "Session Expired!! Please Sign in again.", http.StatusUnauthorized)
			return
		}
		log.Println("session returned: ",session)


		// now get the user profile data for rendering
		up, err := user.FetchUserProfileData(session.Uid)
		if err != nil {
			log.Println("ERROR: Fetching UserProfile for user: "+session.Uid+" Error: "+ err.Error())
			if json == true {
				handleUserProfileJson(rw, req, http.StatusNotFound, up)
				return
			}
			errorHandler(rw, req, "User Not Found.", http.StatusNotFound )
			return
		}

		if json == true {
			handleUserProfileJson(rw, req, http.StatusOK, up)
			return
		}

		profileData, err := Asset("static/profile.html")
		if err != nil {
			panic("profile not found") // must be in bindata
		}
		profileTemplate, err := template.New("profile").Parse(string(profileData))
		if err != nil {
			log.Println("profile template parse failed") // must be valid
			errorHandler(rw, req, "404 Page Not Found", http.StatusNotFound)
			return
		}

		profileBuf := new(bytes.Buffer)
		err = profileTemplate.Execute(profileBuf, up)
		if err != nil {
			errorHandler(rw, req, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		rw.Write(profileBuf.Bytes())

	} else if req.Method == "POST" {
		log.Println("Error: invalid request type: POST")
		errorHandler(rw, req, "Invalid Request", http.StatusBadRequest)
		return
	}
}




