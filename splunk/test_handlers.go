package splunk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func EventTestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	queryParam := r.URL.Query()
	var offset int
	if v, ok := queryParam["offset"]; ok {
		offset, _ = strconv.Atoi(v[0])
	}

	event := &Events{
		Offset: offset,
		Results: []*result{&result{
			Raw: "1 line",
		}}}

	JSON, _ := json.Marshal(event)
	fmt.Fprint(w, string(JSON))
}

func CancelSearchTestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	action := strings.TrimSpace(r.FormValue("action"))
	if action != "cancel" {
		w.WriteHeader(500)
		m := &messages{Messages: []*msg{&msg{
			Type: "WARN",
			Text: fmt.Sprintf("expected cancel action got: %s", action),
		}}}
		JSON, _ := json.Marshal(m)
		fmt.Fprint(w, string(JSON))
	}
	w.Write([]byte("ok"))
}

func LoginTestHandler(w http.ResponseWriter, r *http.Request) {
	sessionKey := "test"
	w.Header().Set("Content-Type", "application/json")
	JSON, _ := json.Marshal(&session{Key: sessionKey})
	fmt.Fprint(w, string(JSON))
}

func SubmitSearchTestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate search query
	query := strings.TrimSpace(r.FormValue("search"))
	expextedQuery := fmt.Sprintf("%s %s", "search", "*")
	if query != expextedQuery {
		w.WriteHeader(500)
		m := &messages{Messages: []*msg{&msg{
			Type: "WARN",
			Text: fmt.Sprintf("expected \"%s\" got \"%s\"", expextedQuery, query),
		}}}
		JSON, _ := json.Marshal(m)
		fmt.Fprint(w, string(JSON))
	}

	// Validate qenerated search ID
	id := strings.TrimSpace(r.FormValue("id"))
	if !strings.HasPrefix(id, "rt_") || !strings.HasSuffix(id, "stailf") {
		w.WriteHeader(500)
		m := &messages{Messages: []*msg{&msg{
			Type: "WARN",
			Text: fmt.Sprintf("Not a valid prefix or suffix for id: %s", id),
		}}}
		JSON, _ := json.Marshal(m)
		fmt.Fprint(w, string(JSON))
	}

	JSON, _ := json.Marshal(&searchJob{Id: id})
	fmt.Fprint(w, string(JSON))
}
