package ledger

import (
	"database/sql"
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

func Balance(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	username := query.Get("username")

	account := findAccount(username)

	fmt.Fprint(w, account.Balance)
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
	if er != nil {
		switch err.(type) {
		case *ErrUserNotFound:
			tryCreatingAccount(username)
			balanceAgain, _ := doTransaction(username, i1)
			fmt.Fprint(w, balanceAgain)
			return
		default:
			fmt.Fprint(w, "Oops! Something techy happened!!")
		}
	} else {
		fmt.Fprint(w, balance)
	}
}

func Withdraw(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	username := query.Get("username")
	amount := query.Get("amount")

	i1, err := strconv.Atoi(amount)
	if err != nil {
		panic(err)
	}

	//negate the amount received in the request
	i1 = i1 * -1

	balance, err := doTransaction(username, i1)

	if err != nil {
		switch err.(type) {
		case *ErrUserNotFound:
			tryCreatingAccount(username)
			balanceAgain, _ := doTransaction(username, i1)
			fmt.Fprint(w, balanceAgain)
			return
		case *ErrInsufficientFunds:
			fmt.Fprint(w, "Insufficient Funds")
			return
		default:
			fmt.Fprint(w, "Oops! Something techy happened!!")
		}
	} else {
		fmt.Fprint(w, balance)
	}
}

func tryCreatingAccount(qualifiedUsername string) {
	sqlStatement := `
	INSERT INTO account (qualified_username, balance)
	VALUES ($1, $2)`
	_, err := db.Exec(sqlStatement, qualifiedUsername, 1000)
	if err != nil {
		panic(err)
	}
}

func doTransaction(qualifiedUsername string, amount int) (float64, error) {
	acc := findAccount(qualifiedUsername)

	if acc == nil {
		return 0, NewErrUserNotFound("User not found")
	}

	sqlStatement := `
		UPDATE account
		SET balance = balance + $1 
		WHERE qualified_username = $2 AND balance + $3 >= 0;`
	res, err := db.Exec(sqlStatement, amount, qualifiedUsername, amount)
	if err != nil {
		panic(err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}

	if count == 0 {
		return 0, NewErrInsufficientFunds("Insufficient funds")
	}

	acc = findAccount(qualifiedUsername)

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

//ErrUserNotFound model for user not found
type ErrUserNotFound struct {
	message string
}

//NewErrUserNotFound function for User not found
func NewErrUserNotFound(message string) *ErrUserNotFound {
	return &ErrUserNotFound{
		message: message,
	}
}
func (e *ErrUserNotFound) Error() string {
	return e.message
}

//ErrInsufficientFunds model for insufficient funds
type ErrInsufficientFunds struct {
	message string
}

//NewErrInsufficientFunds function for insufficient funds
func NewErrInsufficientFunds(message string) *ErrInsufficientFunds {
	return &ErrInsufficientFunds{
		message: message,
	}
}
func (e *ErrInsufficientFunds) Error() string {
	return e.message
}
