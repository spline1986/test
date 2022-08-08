package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

type session struct {
	username string
	expire   time.Time
}

func (s session) isExpired() bool {
	return s.expire.Before(time.Now())
}

var sessions = map[string]session{}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()

		userExists, err := checkUser(r.FormValue("login"), r.FormValue("password"))
		if err != nil || !userExists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sessionToken := uuid.NewString()
		expiredAt := time.Now().Add(86400 * time.Second)

		sessions[sessionToken] = session{
			username: r.FormValue("login"),
			expire:   expiredAt,
		}

		err = login(r.FormValue("login"))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  expiredAt,
			SameSite: http.SameSiteStrictMode,
		})
		toLog(fmt.Sprintf("%s logged in", r.FormValue("login")))
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		r.ParseForm()
		err := logout(r.FormValue("login"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Expires:  time.Unix(0, 0),
			SameSite: http.SameSiteStrictMode,
		})
		toLog(fmt.Sprintf("%s logged out", r.FormValue("login")))
	}
}

func checkSession(w http.ResponseWriter, r *http.Request) (session, bool) {
	c, err := r.Cookie("session_token")
	if err != nil {
		return session{}, false
	}
	sessionToken := c.Value
	userSession, exists := sessions[sessionToken]
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		return session{}, false
	}
	if userSession.isExpired() {
		delete(sessions, sessionToken)
		w.WriteHeader(http.StatusUnauthorized)
		return session{}, false
	}
	return userSession, true
}

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	type sessionStatus struct {
		Username string
		Status   bool
	}

	userSession, ok := checkSession(w, r)
	status := sessionStatus{userSession.username, ok}
	response, _ := json.Marshal(status)
	fmt.Fprintf(w, "%s", response)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	templates, err := template.ParseFiles("templates/index.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	templates.ExecuteTemplate(w, "index.html", nil)
}

func variantsHandler(w http.ResponseWriter, r *http.Request) {
	type variantResponse struct {
		Status   bool
		Variants []Variant
	}
	variants, err := getVariants()
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
		return
	}
	response, _ := json.Marshal(variantResponse{Status: true, Variants: variants})
	fmt.Fprintf(w, "%s", response)
}

func questionHandler(w http.ResponseWriter, r *http.Request) {
	type res struct {
		Next     bool
		Status   bool
		Variant  int
		Question Question
	}
	usersession, ok := checkSession(w, r)
	if !ok {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": \"session error\"}")
		return
	}
	ids := strings.Split(strings.TrimPrefix(r.URL.Path, "/question/"), "/")
	if len(ids) < 2 {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": \"variant or question not found\"}")
		return
	}
	variantId, err := strconv.Atoi(ids[0])
	if err != nil {
		fmt.Fprintf(w, "\"Status\": false, \"Error\": %q}", err)
		return
	}
	questionN, err := strconv.Atoi(ids[1])
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
		return
	}
	next, err := isNextQuestionExists(variantId, questionN)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
		return
	}
	question, err := getQuestion(variantId, questionN)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
		return
	}
	response, _ := json.Marshal(res{Status: true, Variant: variantId, Question: question, Next: next})
	fmt.Fprintf(w, "%s\n", response)
	toLog(fmt.Sprintf("%s get question #%d \"%s\"", usersession.username, question.Id, question.Text))
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	userSession, ok := checkSession(w, r)
	if !ok {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": \"session error\"}")
		return
	}
	variant := strings.TrimPrefix(r.URL.Path, "/start/")
	variantId, err := strconv.Atoi(variant)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
		return
	}
	err = startTest(variantId, userSession.username)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
		return
	}
	questionIds, err := getQuestionsIds(variantId)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
		return
	}
	fmt.Fprintf(w, "{\"Status\": true, \"QuestionId\": "+strconv.Itoa(questionIds[0])+"}")
	toLog(fmt.Sprintf("%s start test variant %s", userSession.username, variant))
}

func saveAnswersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	userSession, ok := checkSession(w, r)
	if !ok {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": \"user not found\"}")
		return
	}
	userId, err := getUserByName(userSession.username)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
	}
	r.ParseForm()
	variantId, err := strconv.Atoi(r.FormValue("variant"))
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
	}
	testId, err := getLastTest(userId, variantId)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
	}
	answers := strings.Split(r.FormValue("answers"), ",")
	var answerId int
	for i := 0; i < len(answers); i++ {
		answerId, err = strconv.Atoi(answers[i])
		if err != nil {
			fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
			return
		}
		err = saveAnswer(testId, answerId)
		if err != nil {
			fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q}", err)
		}
	}
	err = testResult(testId, variantId)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q", err)
		return
	}
	fmt.Fprintf(w, "{\"Status\": true, \"TestId\": %d}", testId)
	toLog(fmt.Sprintf("%s end test variant %d", userSession.username, variantId))
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := checkSession(w, r)
	if !ok {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": \"session error\"}")
		return
	}
	test := strings.TrimPrefix(r.URL.Path, "/result/")
	testId, err := strconv.Atoi(test)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q", err)
		return
	}
	row, err := DB.Query("SELECT percent FROM results WHERE test_id = $1", testId)
	if err != nil {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": %q", err)
		return
	}
	var percent int
	if !row.Next() {
		fmt.Fprintf(w, "{\"Status\": false, \"Error\": \"result not found\"}")
		return
	}
	row.Scan(&percent)
	fmt.Fprintf(w, "{\"Status\": true, \"Result\": %d}", percent)
}

func startHttpServer(port int) {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/session_status", sessionHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/variants", variantsHandler)
	http.HandleFunc("/start/", startHandler)
	http.HandleFunc("/question/", questionHandler)
	http.HandleFunc("/save_answers", saveAnswersHandler)
	http.HandleFunc("/result/", resultHandler)
	fmt.Println("Listen on http://127.0.0.1:" + strconv.Itoa(port) + "/")
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}
