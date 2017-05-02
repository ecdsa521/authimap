package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/mxk/go-imap/imap"
)

var (
	router *httprouter.Router
	c      *imap.Client
	cmd    *imap.Command
	rsp    *imap.Response
	cache  map[string]int
)

func main() {
	cache = make(map[string]int)
	router = httprouter.New()
	router.GET("/", request)
	http.ListenAndServe(":6666", router)
}
func request(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	//one HTTP header has to be passed
	//X-Imap-Backend: imap.example.com

	user, password, hasAuth := r.BasicAuth()
	//user/name was supplied - proceed
	if hasAuth {
		//we need to cache responses to not overload the imap server with bazillion requests.
		if val, ok := cache[user+password]; ok {
			w.WriteHeader(val)
			return
		}
		fmt.Printf("Trying to dial %s and login as %s:%s\n", r.Header["X-Imap-Backend"][0], user, password)
		c, err := imap.Dial(r.Header["X-Imap-Backend"][0])
		defer c.Logout(10 * time.Second)
		if err != nil { //server problem
			cache[user+password] = 403
			w.WriteHeader(403)
			return
		}
		if c.Caps["STARTTLS"] {
			c.StartTLS(nil)
		}

		if c.State() == imap.Login {
			_, err := c.Login(user, password)
			if err == nil {
				//login worked
				cache[user+password] = 200
				w.WriteHeader(200)
			} else {
				cache[user+password] = 403
				w.WriteHeader(403)
			}
		}

	} else {
		//no password supplied - restart with auth request
		w.WriteHeader(401)
	}

}

//Example nginx.conf to go with this

//server {
//	listen 80;
//	root /var/www;
//	location /private/ {
//		auth_request /auth;
//		index index.html;
//	}
//	location /auth {
//		proxy_pass http://127.0.0.1:6666/;
//		proxy_set_header X-Imap-Backend "some.imap.host";
//	}
//}
