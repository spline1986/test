package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"

	_ "github.com/lib/pq"
)

type Variant struct {
	Id   int
	Name string
}

// Answer object.
type Answer struct {
	Id   int
	Text string
}

// Question object.
type Question struct {
	Id      int
	Text    string
	Answers []Answer
	Correct int
}

var DB *sql.DB

// Connecting to PostgreSQL database.
func databaseConnect(host string, username string, password string, database string) {
	psqlconn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, 5432, username, password, database)
	var err error
	DB, err = sql.Open("postgres", psqlconn)
	if err != nil {
		panic(err)
	}
}

// Check is user exists.
func checkUser(username string, password string) (bool, error) {
	row, err := DB.Query("SELECT id FROM users WHERE username = $1 AND password = crypt($2, password)", username, password)
	if err != nil {
		return false, err
	}
	defer row.Close()
	return row.Next(), nil
}

func getUserByName(username string) (int, error) {
	row, err := DB.Query("SELECT id FROM users WHERE username = $1", username)
	if err != nil {
		return -1, err
	}
	defer row.Close()
	if row.Next() {
		var id int
		row.Scan(&id)
		return id, nil
	}
	return -1, errors.New("user not found")
}

func getLastTest(userId int, variantId int) (int, error) {
	row, err := DB.Query("SELECT id FROM test_start WHERE user_id = $1 AND variant_id = $2 ORDER BY id DESC LIMIT 1", userId, variantId)
	if err != nil {
		return -1, err
	}
	defer row.Close()
	if row.Next() {
		var id int
		row.Scan(&id)
		return id, nil
	}
	return -1, errors.New("test start not found")
}

// Write log in information.
func login(username string) error {
	_, err := DB.Exec("INSERT INTO authorizations (username, authorized, login_time) VALUES ($1, true, clock_timestamp())", username)
	return err
}

// Write log out information.
func logout(username string) error {
	_, err := DB.Exec("UPDATE authorizations SET authorized = false, logout_time = clock_timestamp() WHERE username = $1 AND authorized = true", username)
	return err
}

func startTest(variantId int, username string) error {
	row, err := DB.Query("SELECT id FROM users WHERE username = $1", username)
	if err != nil {
		return err
	}
	defer row.Close()
	var userId int
	if !row.Next() {
		return errors.New("user " + username + " not found")
	}
	row.Scan(&userId)
	_, err = DB.Exec("INSERT INTO test_start (user_id, variant_id, start_time) VALUES ($1, $2, clock_timestamp())", userId, variantId)
	return err
}

func isVariantIdExists(id int) (bool, error) {
	row, err := DB.Query("SELECT 1 FROM variants WHERE id = $1", id)
	if err != nil {
		return false, err
	}
	defer row.Close()
	return row.Next(), nil
}

// Get all test variants.
func getVariants() ([]Variant, error) {
	row, err := DB.Query("SELECT id, name FROM variants")
	if err != nil {
		return []Variant{}, err
	}
	defer row.Close()
	var variants []Variant
	var id int
	var text string
	for row.Next() {
		row.Scan(&id, &text)
		variants = append(variants, Variant{id, text})
	}
	return variants, nil
}

// Get id of first question of variant.
func getQuestionsIds(variant int) ([]int, error) {
	var result []int
	variantExists, err := isVariantIdExists(variant)
	if err != nil {
		return result, err
	}
	if !variantExists {
		return result, errors.New("incorrect variant id")
	}
	row, err := DB.Query("SELECT id FROM questions WHERE variant_id = $1 ORDER BY id", variant)
	if err != nil {
		return result, err
	}
	defer row.Close()
	var questionId int
	for row.Next() {
		err = row.Scan(&questionId)
		if err != nil {
			return result, err
		}
		result = append(result, questionId)
	}
	return result, nil
}

func isNextQuestionExists(variant int, question int) (bool, error) {
	questionIds, err := getQuestionsIds(variant)
	if err != nil {
		return false, err
	}
	if question < len(questionIds) {
		return true, nil
	}
	return false, nil
}

// Get question by id.
func getQuestion(variant int, question int) (Question, error) {
	questionIds, err := getQuestionsIds(variant)
	if err != nil {
		return Question{}, err
	}
	if question > len(questionIds) {
		return Question{}, errors.New("incorrect question id")
	}
	row, err := DB.Query("SELECT id, question FROM questions WHERE id = $1 AND variant_id = $2", questionIds[question-1], variant)
	if err != nil {
		return Question{}, err
	}
	defer row.Close()
	if !row.Next() {
		return Question{}, errors.New("cannot read next question")
	}
	var questionId int
	var questionText string
	row.Scan(&questionId, &questionText)
	resultQuestion := Question{Id: questionId, Text: questionText}
	row, err = DB.Query("SELECT id, answer, correct FROM answers_variants WHERE question_id = $1", questionIds[question-1])
	if err != nil {
		return Question{}, err
	}
	defer row.Close()
	var answerId int
	var answer string
	var correct bool
	for row.Next() {
		row.Scan(&answerId, &answer, &correct)
		resultQuestion.Answers = append(resultQuestion.Answers, Answer{answerId, answer})
		if correct {
			resultQuestion.Correct = answerId
		}
	}
	return resultQuestion, nil
}

func saveAnswer(testId int, answerId int) error {
	_, err := DB.Exec("INSERT INTO answers (test_id, answer_id) VALUES ($1, $2)", testId, answerId)
	return err
}

func testResult(testId int, variantId int) error {
	row, err := DB.Query("SELECT answer_id FROM answers WHERE test_id = $1", testId)
	if err != nil {
		return err
	}
	defer row.Close()
	var answerId int
	var correctCount int
	for row.Next() {
		row.Scan(&answerId)
		answerVariant, err := DB.Query("SELECT correct FROM answers_variants WHERE id = $1", answerId)
		if err != nil {
			return err
		}
		var correct bool
		if !answerVariant.Next() {
			return errors.New("answer variant not found " + strconv.Itoa(answerId))
		}
		answerVariant.Scan(&correct)
		if correct {
			correctCount++
		}
	}
	row, err = DB.Query("SELECT count(1) FROM questions WHERE variant_id = $1", variantId)
	if err != nil {
		return err
	}
	row.Next()
	var answerCount int
	row.Scan(&answerCount)
	percent := int(math.Round(float64(correctCount) / float64(answerCount) * 100))
	_, err = DB.Exec("INSERT INTO results (test_id, percent) VALUES ($1, $2)", testId, percent)
	if err != nil {
		return err
	}
	return nil
}
