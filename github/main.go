package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// Fake database user in our application
// Key is github ID, value is user ID
var userDatabase = map[string]string{}

// Github response
// JSON layout: {"data":{"viewer":{"id":"..."}}}
type githubResponse struct {
	Data struct {
		Viewer struct {
			ID string `json:"id"`
		} `json:"viewer"`
	} `json:"data"`
}

var githubOauthConfig = &oauth2.Config{}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println(err)
		return
	}

	githubOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		Endpoint:     github.Endpoint,
		// RedirectURL: , << if empty, use value on github oauth (http://localhost:8080/oauth2/receive)
	}

	userDatabase["MDQ6VXNlcjM5ODM1OTYy"] = "muchlis-123"

	http.HandleFunc("/", index)
	http.HandleFunc("/oauth/github", startGithubOauth)
	http.HandleFunc("/oauth2/receive", completeGithubOauth)
	fmt.Println("server started, port:  8080")
	http.ListenAndServe(":8080", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Document</title>
</head>
<body>
	<form action="/oauth/github" method="post">
		<input type="submit" value="Login with Github">
	</form>
</body>
</html>`)
}

func startGithubOauth(w http.ResponseWriter, r *http.Request) {
	redirectURL := githubOauthConfig.AuthCodeURL("0000")
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func completeGithubOauth(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	state := r.FormValue("state")

	if state != "0000" {
		http.Error(w, "State is incorrect", http.StatusBadRequest)
		return
	}

	token, err := githubOauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Couldn't login", http.StatusInternalServerError)
		return
	}

	ts := githubOauthConfig.TokenSource(r.Context(), token)
	client := oauth2.NewClient(r.Context(), ts)

	requestBody := strings.NewReader(`{"query": "query {viewer {id}}"}`)
	resp, err := client.Post("https://api.github.com/graphql", "application/json", requestBody)
	if err != nil {
		http.Error(w, "Couldn't get user", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var gr githubResponse
	err = json.NewDecoder(resp.Body).Decode(&gr)
	if err != nil {
		http.Error(w, "Github invalid response", http.StatusInternalServerError)
		return
	}

	githubID := gr.Data.Viewer.ID
	fmt.Println("github_id: ", githubID)
	userID, ok := userDatabase[githubID]
	if !ok {
		// New User - create account
		// Maybe return, maybe not, depends
		fmt.Println("new user has login")
	}

	fmt.Println(userID)
	// Generate JWT with role
	// Login to account userID using JWT
}
