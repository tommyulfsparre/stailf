package splunk

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSplunkLogin(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(LoginTestHandler))
	defer ts.Close()

	client := NewClient("admin", "admin", ts.URL)
	err := client.Login()
	if err != nil {
		t.Fatal(err)
	}

	sessionKey := "test"
	if client.session.Key != sessionKey {
		t.Fatalf("expected sessionKey:%s got %s\n", sessionKey, client.session.Key)
	}
}

func TestCancelSearch(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(CancelSearchTestHandler))
	defer ts.Close()

	client := NewClient("admin", "admin", ts.URL)
	err := client.CancelSearch("12345.stailf")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSplunkSubmitSearch(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SubmitSearchTestHandler))
	defer ts.Close()

	client := NewClient("admin", "admin", ts.URL)
	params := Param{
		"search_mode": "realtime",
	}

	searchQuery := "*"
	id_suffix := "stailf"
	sid, err := client.SubmitSearch(searchQuery, id_suffix, params)
	if err != nil {
		t.Fatal(err)
	}

	if sid == "" {
		t.Fatal("SearchJob didn't return a sid")
	}
}

func TestSplunkEvents(t *testing.T) {

	expectedOffset := 999

	ts := httptest.NewServer(http.HandlerFunc(EventTestHandler))
	defer ts.Close()

	client := NewClient("admin", "admin", ts.URL)
	event, err := client.Events("12345.stilf", expectedOffset)
	if err != nil {
		t.Fatal(err)
	}

	if event.Offset != expectedOffset {
		t.Fatalf("expected offset: %d got %d\n", expectedOffset, event.Offset)
	}
}

func TestSplunkStreamEvents(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(EventTestHandler))
	defer ts.Close()

	client := NewClient("admin", "admin", ts.URL)
	eventCh := client.StreamEvents("12345.stailf")

	var results int
	for event := range eventCh {
		for _ = range event.(*Events).Results {
			results += 1
		}
		client.Shutdown()
	}

	if results != 2 {
		t.Fatalf("expected %d results got %d", 1, results)
	}
}
