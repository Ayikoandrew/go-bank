package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	UpdateAccount(*Account) error
	DeleteAccount(int) error
	GetAccountByID(int) (*Account, error)
	GetAccounts() ([]*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := "host=localhost port=32768 user=postgres dbname=bank password=gobank sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Init() error {
	return s.createAcountTable()
}

func (s *PostgresStore) createAcountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account (
	id SERIAL PRIMARY KEY,
	first_name VARCHAR(255),
	last_name VARCHAR(255),
	number bigserial,
	balance FLOAT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := s.db.Exec(query)
	return err
}

func (p *PostgresStore) CreateAccount(account *Account) error {
	query := `insert into account (first_name, last_name, number, balance, created_at) values ($1, $2, $3, $4, $5)`
	resp, err := p.db.Query(
		query,
		account.FirstName,
		account.LastName,
		account.Number,
		account.Balance,
		account.CreatedAt)
	if err != nil {
		return err
	}

	fmt.Printf("%+v\n", resp)
	return nil
}

func (p *PostgresStore) UpdateAccount(account *Account) error {
	query := `update account set first_name = $1, last_name = $2, number = $3, balance = $4, created_at = $5 where id = $6`
	_, err := p.db.Query(query, account.FirstName, account.LastName, account.Number, account.Balance, account.CreatedAt, account.ID)

	if err != nil {
		return err
	}
	return nil
}

func (p *PostgresStore) DeleteAccount(id int) error {
	_, err := p.db.Query("delete from account where id = $1", id)
	if err != nil {
		return err
	}
	return nil
}

func (p *PostgresStore) GetAccountByID(id int) (*Account, error) {
	row, err := p.db.Query("select * from account where id = $1", id)

	if err != nil {
		return nil, err
	}
	for row.Next() {
		return scanIntoAccount(row)

	}
	return nil, fmt.Errorf("account with id %d not found", id)
}

func (p *PostgresStore) GetAccounts() ([]*Account, error) {
	query := `select * from account`
	rows, err := p.db.Query(query)

	if err != nil {
		return nil, err
	}

	accounts := []*Account{}

	for rows.Next() {
		account := new(Account)
		account, err := scanIntoAccount(rows)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

func scanIntoAccount(rows *sql.Rows) (*Account, error) {
	account := new(Account)

	if err := rows.Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Number,
		&account.Balance,
		&account.CreatedAt,
	); err != nil {

		return nil, err
	}

	return account, nil
}
