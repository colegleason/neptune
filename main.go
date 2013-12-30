package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strings"

	"neptune/pkgs/user"
	"neptune/pkgs/codify"
	"neptune/pkgs/cookies"
)

type Page struct {
	Title string
	Body  template.HTML
	UserData   template.HTML
}

// Loads a page for use
func loadPage(title string, r *http.Request) (*Page, error) {

	var filename string
	var usr []byte
	var option[]byte

	if cookies.IsLoggedIn(r) {
		cookie, _ := r.Cookie("SessionID")
		z := strings.Split(cookie.Value, ":")
		filename = "accounts/" + z[0] + ".txt"
		usr, _ = ioutil.ReadFile(filename)
		option = []byte("<a href='/logout'>logout</a>")
	} 

	filename = "web/" + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return &Page{Title: title, Body: template.HTML(body), UserData: (template.HTML(usr) + template.HTML(option))}, nil
}

// Shows a particular page
func viewHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/"):]
	p, err := loadPage(title, r)

	if err != nil && !cookies.IsLoggedIn(r) {
		http.Redirect(w, r, "/home", http.StatusFound)
		return
	}else if err != nil{
		http.Redirect(w, r, "/login-succeeded", http.StatusFound)
		return
	}

	t, _ := template.ParseFiles("view.html")
	t.Execute(w, p)
}

// Handles the users loggin and gives them a cookie for doing so
func loginHandler(w http.ResponseWriter, r *http.Request) {
	usr := new(user.User)
	usr.Email = r.FormValue("email")
	pass := r.FormValue("pwd")

	if len(pass) > 0 {
		usr.Password = codify.SHA(pass)
		ok := user.CheckCredentials(usr.Email, usr.Password)
		if ok {
			user.CreateUserFile(usr.Email)
			cookie := cookies.LoginCookie(usr.Email)
			http.SetCookie(w, &cookie)
			usr.SessionID = cookie.Value	
			_ = user.UpdateUser(usr)
			http.Redirect(w, r, "/login-succeeded", http.StatusFound)
		} else {
			http.Redirect(w, r, "/login-failed", http.StatusFound)
		}
	} else {
		viewHandler(w, r)
	}
}

// Registers the new user
func registerHandler(w http.ResponseWriter, r *http.Request) {
	usr := new(user.User)
	usr.Email = r.FormValue("email")
	pass := r.FormValue("pwd")
	
	if len(pass) > 0 {
		usr.Password = codify.SHA(pass)
		if user.DoesAccountExist(usr.Email) {
			http.Redirect(w, r, "/account-exists", http.StatusFound)
		} else {
			ok := user.CreateAccount(usr)
			if ok {
				http.Redirect(w, r, "/success", http.StatusFound)
			} else {
				http.Redirect(w, r, "/failed", http.StatusFound)
			}
		}
	} else {
		viewHandler(w, r)
	}
}

// Logs out the user, removes their cookie from the database
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	
	cookie, err := r.Cookie("SessionID")
	if err != nil {
		fmt.Println(err)
		return
	}

	sessionID := cookie.Value

	z := strings.Split(sessionID, ":")
	usr := new(user.User)
	usr.Email = z[0]
	usr.SessionID = z[0] + ":" + z[1]

	session, err := mgo.Dial("127.0.0.1:27017/")
	if err != nil {
		return
	}
	c := session.DB("test").C("users")
	fmt.Println(usr.SessionID)
	c.Remove(bson.M{"SessionID": usr.SessionID})

	http.Redirect(w, r, "/home", http.StatusFound)

}

func main() {
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/", viewHandler)
	http.ListenAndServe(":8080", nil)
}
