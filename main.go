package ledger

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

var (
	db *sql.DB

	connectionName = os.Getenv("POSTGRES_INSTANCE_CONNECTION_NAME")
	dbUser         = os.Getenv("POSTGRES_USER")
	dbPassword     = os.Getenv("POSTGRES_PASSWORD")
	dsn            = fmt.Sprintf("user=%s password=%s host=/cloudsql/%s", dbUser, dbPassword, connectionName)
)

func init() {
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Could not open db: %v", err)
	}

	// Only allow 1 connection to the database to avoid overloading it.
	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(1)
}

// Account domain object ledger
type Account struct {
	QualifiedUsername string
	Balance           float64
}

func Deposit(w http.ResponseWriter, req *http.Request) {
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

func Withdraw(w http.ResponseWriter, req *http.Request) {
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
		fmt.Fprint(w, errors.New("Insufficient Funds!!"))
	}

	fmt.Fprint(w, balanceAgain)
}

func Balance(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	username := query.Get("username")

	account := findAccount(username)

	fmt.Fprint(w, account.Balance)
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
