package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Fake database user in our application
// Key is github ID, value is user ID
var userDatabase = map[string]string{}

// global variable level package for config
var googleOauthConfig = &oauth2.Config{}

// example response
//
//	{
//		"id": "108212271843429487853",
//		"email": "whois.muchlis@gmail.com",
//		"verified_email": true,
//		"picture": "https://lh3.googleusercontent.com/a-/ACNPEu9pQoK33mJNHSJUQ8YT6X5gY1W3-OrYyDjneEYmYQ=s96-c"
//	}
type GoogleResponse struct {
	ID            string `json:"id,omitempty"`
	Email         string `json:"email,omitempty"`
	VerifiedEmail bool   `json:"verified_email,omitempty"`
	Picture       string `json:"picture,omitempty"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
		return
	}

	// inject config for oauth2
	// here we can supply config from domain management
	googleOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080/application1/oauth2/receive",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
	}

	// inject user existing in fake database
	userDatabase["whois.muchlis@gmail.com"] = "muchlis-123"

	// handler
	http.HandleFunc("/", index)
	http.HandleFunc("/oauth/google", startGoogleOauth)
	http.HandleFunc("/application1/oauth2/receive", completeGoogleOauth)
	fmt.Println("server started, port:  8080")
	http.ListenAndServe(":8080", nil)
}

// template
func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Document</title>
</head>
<body>
	<form action="/oauth/google?application=aplication1" method="post">
		<input type="submit" value="Login with Google Application1">
	</form>
	<form action="/oauth/google?application=aplication2" method="post">
		<input type="submit" value="Login with Google Application2">
	</form>
</body>
</html>`)
}

func startGoogleOauth(w http.ResponseWriter, r *http.Request) {
	application := r.FormValue("application")

	// application destination from queryparams with state
	// in here we can check to domain management data for list application
	state := ""
	switch application {
	case "aplication1":
		state = "0000-aplication1"
	case "aplication2":
		state = "0000-aplication2"
	default:
		state = "0000"
	}

	redirectURL := googleOauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func completeGoogleOauth(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	state := r.FormValue("state")

	// get application domain from state
	applicationName, err := getApplicationNameFromState(state)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println("run call calback ", applicationName)

	token, err := googleOauthConfig.Exchange(r.Context(), code)
	if err != nil {
		fmt.Println("failed exchange for token")
		http.Error(w, "Couldn't login", http.StatusInternalServerError)
		return
	}

	ts := googleOauthConfig.TokenSource(r.Context(), token)
	client := oauth2.NewClient(r.Context(), ts)

	// response, err := client.Get(oauthGoogleUrlAPI + token.AccessToken)
	response, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed getting user info: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	// contents, err := io.ReadAll(response.Body)
	// if err != nil {
	// 	http.Error(w, fmt.Sprintf("failed read response: %s", err.Error()), http.StatusInternalServerError)
	// 	return
	// }

	fmt.Println("completed call calback ", applicationName)

	var gr GoogleResponse
	err = json.NewDecoder(response.Body).Decode(&gr)
	if err != nil {
		http.Error(w, "Github invalid response", http.StatusInternalServerError)
		return
	}

	googleEmail := gr.Email
	fmt.Println("google email: ", googleEmail)
	userID, ok := userDatabase[googleEmail]
	if !ok {
		// New User - create account
		// Maybe return, maybe not, depends
		fmt.Println("new user has login")
	} else {
		fmt.Println("user found: ", userID)
	}

	// TODO:
	// Generate JWT
	// before that we must check all role, permission, what application can be accesses
	// with this unique id user (email)
	// then, response with JWT

	fmt.Fprintf(w, "UserInfo: %v\n", gr)
}

// helper method for parsing application destination from state string
func getApplicationNameFromState(state string) (string, error) {
	stateSplit := strings.Split(state, "-")
	if len(stateSplit) != 2 || stateSplit[0] != "0000" {
		return "", fmt.Errorf("invalid state format")
	}
	return stateSplit[1], nil
}
