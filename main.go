package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type githubAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

var ghresp githubAccessTokenResponse

// init() executes before the main program
func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found")
	}
}

func main() {

	//
	// Root route
	// Simply returns a link to the login route
	http.HandleFunc("/", rootHandler)

	// Login route
	http.HandleFunc("/login/github/", githubLoginHandler)

	// Github callback
	http.HandleFunc("/login/github/callback", githubCallbackHandler)

	// Route where the authenticated user is redirected to
	http.HandleFunc("/loggedin", func(w http.ResponseWriter, r *http.Request) {
		loggedinHandler(w, r, "")
	})
	// Login route
	http.HandleFunc("/pullrequest", pullRequestHandler)

	// Listen and serve on port 3000
	fmt.Println("[ UP ON PORT 3000 ]")
	log.Panic(
		http.ListenAndServe(":3000", nil),
	)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<a href="/login/github/">LOGIN</a>`)
}

func createBranchHandler(w http.ResponseWriter, r *http.Request, refHash string) {
	currentBranch := fmt.Sprintf("refs/heads/test_%s", time.Now())
	postBody, _ := json.Marshal(map[string]string{
		"ref": currentBranch,
		"sha": refHash,
	})
	responseBody := bytes.NewBuffer(postBody)
	refBranch, reqerr := http.NewRequest("POST", "https://api.github.com/repos/jaybatra26/githubauth/git/refs", responseBody)
	token_ := fmt.Sprintf("token %s", ghresp.AccessToken)

	refBranch.Header.Set("Authorization", token_)
	//Handle Error
	if reqerr != nil {
		log.Fatalf("An Error Occured %v", reqerr)
	}
	resp, resperr := http.DefaultClient.Do(refBranch)
	if resperr != nil {
		log.Panic((resperr))
		log.Panic("Request failed")
	}
	log.Panic((resp))
	output_string := fmt.Sprintf("Branch %s created", currentBranch)
	if resp.StatusCode == http.StatusOK {
		log.Panic((output_string))

		fmt.Fprintf(w, output_string)
		return
	}
}

//func pullRequestHandler
func pullRequestHandler(w http.ResponseWriter, r *http.Request) {

	accessToken := ghresp.AccessToken
	if accessToken == "" {
		fmt.Fprintf(w, "UNAUTHORIZED!")
		return
	}

	refHead, reqerr := http.NewRequest("GET", "https://api.github.com/repos/jaybatra26/githubauth/git/refs/heads", nil)
	token_ := fmt.Sprintf("token %s", accessToken)

	refHead.Header.Set("Authorization", token_)

	if reqerr != nil {
		log.Panic("API Request creation failed")
	}
	resp, resperr := http.DefaultClient.Do(refHead)
	if resperr != nil {
		log.Panic((resperr))
		log.Panic("Request failed")
	}
	if resp.StatusCode == http.StatusOK {
		refHash, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(refHash)
		createBranchHandler(w, r, bodyString)
	}
}

// func get repo Handler

func repoHandler(w http.ResponseWriter, r *http.Request) {
	req, reqerr := http.NewRequest("GET", "https://api.github.com/user/repos", nil)
	token_ := fmt.Sprintf("token %s", ghresp.AccessToken)

	req.Header.Set("Authorization", token_)

	if reqerr != nil {
		log.Panic("API Request creation failed")
	}
	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		log.Panic((resperr))
		log.Panic("Request failed")
	}
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		// Prettifying the json
		var prettyJSON bytes.Buffer
		// json.indent is a library utility function to prettify JSON indentation
		parserr := json.Indent(&prettyJSON, []byte(bodyString), "", "\t")
		if parserr != nil {
			log.Panic("JSON parse error")
		}

		// Return the prettified JSON as a string

		fmt.Fprintf(w, string(prettyJSON.Bytes()))
		// fmt.Fprintf(w, bodyString)
	}
}
func loggedinHandler(w http.ResponseWriter, r *http.Request, githubAccessToken string) {
	if githubAccessToken == "" {
		// Unauthorized users get an unauthorized message
		fmt.Fprintf(w, "UNAUTHORIZED!")
		return
	}

	w.Header().Set("Content-type", "application/json")

	repoHandler(w, r)
	//call get Repo handler
}

func githubLoginHandler(w http.ResponseWriter, r *http.Request) {
	githubClientID := getGithubClientID()

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s", githubClientID, "http://localhost:3000/login/github/callback")

	http.Redirect(w, r, redirectURL, 301)
}

func githubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	githubAccessToken := getGithubAccessToken(code)

	loggedinHandler(w, r, githubAccessToken)
}

func getGithubData(accessToken string) string {
	req, reqerr := http.NewRequest("GET", "https://api.github.com/user", nil)
	if reqerr != nil {
		log.Panic("API Request creation failed")
	}

	authorizationHeaderValue := fmt.Sprintf("token %s", accessToken)
	req.Header.Set("Authorization", authorizationHeaderValue)

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		log.Panic("Request failed")
	}

	respbody, _ := ioutil.ReadAll(resp.Body)

	return string(respbody)
}

func getGithubAccessToken(code string) string {

	clientID := getGithubClientID()
	clientSecret := getGithubClientSecret()

	requestBodyMap := map[string]string{"client_id": clientID, "client_secret": clientSecret, "code": code}
	requestJSON, _ := json.Marshal(requestBodyMap)

	req, reqerr := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(requestJSON))
	if reqerr != nil {
		log.Panic("Request creation failed")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		log.Panic("Request failed")
	}

	respbody, _ := ioutil.ReadAll(resp.Body)

	// Represents the response received from Github

	json.Unmarshal(respbody, &ghresp)
	return ghresp.AccessToken
}

func getGithubClientID() string {

	githubClientID, exists := os.LookupEnv("CLIENT_ID")
	if !exists {
		log.Fatal("Github Client ID not defined in .env file")
	}

	return githubClientID
}

func getGithubClientSecret() string {

	githubClientSecret, exists := os.LookupEnv("CLIENT_SECRET")
	if !exists {
		log.Fatal("Github Client ID not defined in .env file")
	}

	return githubClientSecret
}
