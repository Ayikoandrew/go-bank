package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

type ServerAPI struct {
	listenAddr string
	store      Storage
}

func NewServerAPI(listenAddr string, store Storage) *ServerAPI {
	return &ServerAPI{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *ServerAPI) run() {
	router := mux.NewRouter()

	router.HandleFunc("/account", makeHTTPHandlerFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", makeHTTPHandlerFunc(s.handleGetAccountById))
	router.HandleFunc("/delete/{id}", makeHTTPHandlerFunc(s.handleDeleteAccount))
	router.HandleFunc("/transfer", makeHTTPHandlerFunc(s.handleTransfer))

	log.Println("JSON API server is running on", s.listenAddr)

	srv := &http.Server{
		Addr:           s.listenAddr,
		Handler:        router,
		WriteTimeout:   15 * time.Second,
		ReadTimeout:    15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	serverError := make(chan error, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			serverError <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverError:
		log.Printf("server error: %v", err)
	case sig := <-stop:
		log.Printf("Received shutdown signal: %v", sig)
	}

	log.Println("Shutting down the server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed: %v", err)
		return
	}
}

func (s *ServerAPI) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == http.MethodGet {
		return s.handleGetAccount(w, r)
	}

	if r.Method == http.MethodPost {
		return s.handleCreateAccount(w, r)
	}

	if r.Method == http.MethodDelete {
		return s.handleDeleteAccount(w, r)
	}

	if r.Method == http.MethodPut {
		return s.handleUpdateAccount(w, r)
	}
	return fmt.Errorf("unsupported method %s", r.Method)
}

func (s *ServerAPI) handleUpdateAccount(_ http.ResponseWriter, r *http.Request) error {
	account := new(Account)
	if err := json.NewDecoder(r.Body).Decode(account); err != nil {
		return err
	}
	defer r.Body.Close()

	return s.store.UpdateAccount(account)
}

func (s *ServerAPI) handleGetAccountById(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)

	if err != nil {
		return err
	}

	if r.Method == http.MethodGet {

		account, err := s.store.GetAccountByID(id)
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, account)
	} else {
		s.store.DeleteAccount(id)
	}
	return nil
}

func (s *ServerAPI) handleGetAccount(w http.ResponseWriter, _ *http.Request) error {
	account, err := s.store.GetAccounts()
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, account)
}

func (s *ServerAPI) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return err
	}

	account := NewAccount(createAccountReq.FirstName, createAccountReq.LastName)

	if err := s.store.CreateAccount(account); err != nil {
		return err
	}
	return writeJSON(w, http.StatusCreated, account)
}

func (s *ServerAPI) handleDeleteAccount(_ http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}
	return s.store.DeleteAccount(id)

}

func (s *ServerAPI) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	trasferReq := new(TransferRequest)
	if err := json.NewDecoder(r.Body).Decode(trasferReq); err != nil {
		return err
	}
	defer r.Body.Close()

	return writeJSON(w, http.StatusOK, trasferReq)
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

type apiError struct {
	Error string `json:"error"`
}

type apiFunc func(http.ResponseWriter, *http.Request) error

func makeHTTPHandlerFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		}
	}
}

func getID(r *http.Request) (int, error) {
	idStr := mux.Vars(r)
	id, err := strconv.Atoi(idStr["id"])
	if err != nil {
		return id, err
	}

	return id, nil
}
