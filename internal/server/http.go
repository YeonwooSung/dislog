package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func NewHTTPServer(addr string) *http.Server {
	logsrv := newLogServer()
	r := mux.NewRouter()
	r.HandleFunc("/", logsrv.handleProduce).Methods("POST")
	r.HandleFunc("/", logsrv.handleConsume).Methods("GET")
	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

//---------------------------------------------
// custom types

type logServer struct {
	Log *Log
}

type ProduceRequest struct {
	Record Record `json:"record"`
}

type ProduceResponse struct {
	Offset uint64 `json:"offset"`
}

type ConsumeRequest struct {
	Offset uint64 `json:"offset"`
}

type ConsumeResponse struct {
	Record Record `json:"record"`
}

//---------------------------------------------
// Log Server

func newLogServer() *logServer {
	return &logServer{
		Log: NewLog(),
	}
}

//---------------------------------------------
// Produce

func (s *logServer) handleProduce(w http.ResponseWriter, r *http.Request) {
	var req ProduceRequest

	// Decode the request body into the ProduceRequest struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Append the record to the log
	off, err := s.Log.Append(req.Record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// produce the response instance
	res := ProduceResponse{Offset: off}

	// Encode the ProduceResponse struct into JSON and write it to the response body
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//---------------------------------------------
// Consume

func (s *logServer) handleConsume(w http.ResponseWriter, r *http.Request) {
	var req ConsumeRequest

	// Decode the request body into the ConsumeRequest struct
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Read the record from the log
	record, err := s.Log.Read(req.Offset)
	if err == ErrOffsetNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// produce the response instance
	res := ConsumeResponse{Record: record}

	// Encode the ConsumeResponse struct into JSON and write it to the response body
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
