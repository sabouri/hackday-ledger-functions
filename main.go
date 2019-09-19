package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

var db *sql.DB

const (
	dbhost = "DBHOST"
	dbport = "DBPORT"
	dbuser = "DBUSER"
	dbpass = "DBPASS"
	dbname = "DBNAME"
)

// Account domain object ledger
type Account struct {
	QualifiedUsername string
	Balance           float64
}

func deposit(w http.ResponseWriter, req *http.Request) {
	initDb()
	defer db.Close()

	query := req.URL.Query()

	username := query.Get("username")
	amount := query.Get("amount")

	/** converting the str1 variable into an int using Atoi method */
	i1, err := strconv.Atoi(amount)
	if err != nil {
		panic(err)
	}

	balance, er := doTransaction(username, i1)

	if er == nil {
		fmt.Fprint(w, balance)
		return
	}

	tryCreatingAccount(username)

	balance, er = doTransaction(username, i1)

	fmt.Fprint(w, balance)
}

func withdraw(w http.ResponseWriter, req *http.Request) {
	initDb()
	defer db.Close()

	query := req.URL.Query()

	username := query.Get("username")
	amount := query.Get("amount")

	/** converting the str1 variable into an int using Atoi method */
	i1, err := strconv.Atoi(amount)
	if err != nil {
		panic(err)
	}

	i1 = i1 * -1

	balance, er := doTransaction(username, i1)

	if er == nil {
		fmt.Fprint(w, balance)
		return
	}

	tryCreatingAccount(username)

	balanceAgain, er1 := doTransaction(username, i1)

	if er1 != nil {
		fmt.Fprint(w, errors.New("Insufficient Funds"))
	}

	fmt.Fprint(w, balanceAgain)
}

func balance(w http.ResponseWriter, req *http.Request) {
	initDb()
	defer db.Close()

	query := req.URL.Query()

	username := query.Get("username")

	account := findAccount(username)

	fmt.Fprint(w, account.Balance)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/balance", balance)
	mux.HandleFunc("/deposit", deposit)
	mux.HandleFunc("/withdraw", withdraw)

	http.ListenAndServe(":8080", mux)
}

func initDb() {
	config := dbConfig()
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		config[dbhost], config[dbport],
		config[dbuser], config[dbpass], config[dbname])

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected!")
}

func dbConfig() map[string]string {
	conf := make(map[string]string)
	host, ok := os.LookupEnv(dbhost)
	if !ok {
		panic("DBHOST environment variable required but not set")
	}
	port, ok := os.LookupEnv(dbport)
	if !ok {
		panic("DBPORT environment variable required but not set")
	}
	user, ok := os.LookupEnv(dbuser)
	if !ok {
		panic("DBUSER environment variable required but not set")
	}
	password, ok := os.LookupEnv(dbpass)
	if !ok {
		panic("DBPASS environment variable required but not set")
	}
	name, ok := os.LookupEnv(dbname)
	if !ok {
		panic("DBNAME environment variable required but not set")
	}
	conf[dbhost] = host
	conf[dbport] = port
	conf[dbuser] = user
	conf[dbpass] = password
	conf[dbname] = name
	return conf
}

func tryCreatingAccount(qualifiedUsername string) {
	sqlStatement := `
	INSERT INTO account (qualified_username, balance)
	VALUES ($1, $2)`
	_, err := db.Exec(sqlStatement, qualifiedUsername, 100000)
	if err != nil {
		panic(err)
	}
}

func doTransaction(qualifiedUsername string, amount int) (float64, error) {
	account := findAccount(qualifiedUsername)
	modifiedBalance := account.Balance + float64(amount)
	sqlStatement := `
		UPDATE account
		SET balance = $1 
		WHERE qualified_username = $2 AND balance > $3;`
	res, err := db.Exec(sqlStatement, modifiedBalance, qualifiedUsername, modifiedBalance)
	if err != nil {
		panic(err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}

	if int(count) == 0 {
		return 0, errors.New("this is an error")
	}

	acc := findAccount(qualifiedUsername)

	return acc.Balance, nil
}

func findAccount(qualifiedUsername string) *Account {
	sqlStatement := `SELECT qualified_username,balance FROM account WHERE qualified_username=$1;`
	acc := new(Account)
	row := db.QueryRow(sqlStatement, qualifiedUsername)
	err := row.Scan(&acc.QualifiedUsername, &acc.Balance)
	switch err {
	case sql.ErrNoRows:
		return nil
	case nil:
		return acc
	default:
		panic(err)
	}
}
