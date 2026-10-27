package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"sync"
	"unicode/utf8"

	"genanalex/internal/sql"
)

// sqlDataDir is the database the web page works on. The client never chooses
// the path, so requests cannot read or write outside it.
var sqlDataDir = sql.DefaultDir()

// sqlMu serialises runs: each one rewrites metadata and table files.
var sqlMu sync.Mutex

type sqlRequest struct {
	Source  string `json:"source"`
	Execute bool   `json:"execute"`
}

func handleSQLRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		writeErr(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxCompiscriptRequestBytes+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(body) > maxCompiscriptRequestBytes {
		writeErr(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	if !utf8.Valid(body) {
		writeErr(w, http.StatusBadRequest, "request body must be valid UTF-8")
		return
	}
	var req sqlRequest
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	sqlMu.Lock()
	rep := (&sql.Engine{Dir: sqlDataDir}).Run(req.Source, req.Execute)
	sqlMu.Unlock()
	json.NewEncoder(w).Encode(rep)
}
