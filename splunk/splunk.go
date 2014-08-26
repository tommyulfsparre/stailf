package splunk

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SplunkClient is used to talk to Splunk's REST api
type SplunkClient struct {
	baseURL  string
	username string
	password string
	stopped  bool
	session  session
	client   *http.Client
	stopCh   chan struct{}
}

type Events struct {
	Offset   int       `json:"init_offset"`
	Messages []*msg    `json:"messages"`
	Results  []*result `json:"results"`
}

type result struct {
	ConfStr       string `json:"_confstr"`
	IndexTime     string `json:"_indextime"`
	Raw           string `json:"_raw"`
	Serial        string `json:"_serial"`
	SourceTypeRaw string `json:"_sourcetype"`
	Time          string `json:"_time"`
	Host          string `json:"host"`
	Index         string `json:"index"`
	LineCount     string `json:"linecount"`
	Source        string `json:"source"`
	SourceType    string `json:"sourcetype"`
	SplunkServer  string `json:"splunk_server"`
}

type messages struct {
	Messages []*msg `json:"messages"`
}

type msg struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type searchJob struct {
	Id string `json:"sid"`
}

type session struct {
	Key string `json:"sessionKey"`
}

type Param map[string]interface{}

// NewClient will return a new Splunk client
func NewClient(username, password, url string) *SplunkClient {
	return &SplunkClient{
		baseURL:  url,
		username: username,
		password: password,
		client:   newInsecureHttpClient(url),
		stopCh:   make(chan struct{}),
	}
}

// send sends a request to Splunk with the supplied method and parameters.
// send handles Unmarshalling of the JSON response
func (sc *SplunkClient) send(method, url string, params url.Values, v interface{}) error {

	if method == "GET" {
		url = fmt.Sprintf("%s?%s", url, params.Encode())
		params = nil
	}

	// Build request
	req, err := sc.newRequest(method, url, params)
	if err != nil {
		return err
	}
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Send request
	resp, err := sc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle non 2xx http response codes
	if err = isStatusOK(resp); err != nil {
		return err
	}

	// Read JSON response
	if v != nil {
		err = readJSON(resp, v)
		if err != nil {
			return err
		}
	} else {
		// Read and throw away the response
		io.Copy(ioutil.Discard, resp.Body)
	}

	return nil
}

// newRequest will return a http.Request with HTTP Basic Auth set to either
// the username:password or Splunk:sessionKey
func (sc *SplunkClient) newRequest(method, url string, data url.Values) (req *http.Request, err error) {

	req, err = http.NewRequest(method, url, strings.NewReader(data.Encode()))
	if err != nil {
		return req, err
	}

	if sc.session.Key != "" {
		auth := fmt.Sprintf("%s %s", "Splunk", sc.session.Key)
		req.Header.Set("Authorization", auth)
	} else {
		req.SetBasicAuth(sc.username, sc.password)
	}

	return req, err
}

// Login will auth to Splunk and obtain a sessionKey
func (sc *SplunkClient) Login() error {

	data := url.Values{
		"username":    {sc.username},
		"password":    {sc.password},
		"output_mode": {"json"},
	}

	loginURL := fmt.Sprintf("%s/services/auth/login", sc.baseURL)
	var s session
	err := sc.send("POST", loginURL, data, &s)
	if err != nil {
		return err
	}

	sc.session = s
	return nil
}

// CancelSearch will cancel a search job
func (sc *SplunkClient) CancelSearch(sid string) error {

	// Required values
	data := url.Values{
		"action":      {"cancel"},
		"output_mode": {"json"},
	}

	controlURL := fmt.Sprintf("%s/services/search/jobs/%s/control", sc.baseURL, sid)
	err := sc.send("POST", controlURL, data, nil)
	if err != nil {
		return err
	}

	return nil
}

// SubmitSearch will create a new search job
func (sc *SplunkClient) SubmitSearch(query, suffix string, params Param) (sid string, err error) {

	// The search string for the search parameter must be prefixed with "search."
	if !strings.HasPrefix(query, "search") {
		query = strings.Join([]string{"search", query}, " ")
	}

	// Generate a unique ID for a new search job
	id, err := genID(params, suffix)
	if err != nil {
		return sid, err
	}

	// Required values
	data := url.Values{
		"id":          {id},
		"search":      {query},
		"output_mode": {"json"},
	}
	for k, v := range params {
		data.Add(k, fmt.Sprintf("%v", v))
	}

	jobsURL := fmt.Sprintf("%s/services/search/jobs", sc.baseURL)
	var s searchJob
	err = sc.send("POST", jobsURL, data, &s)
	if err != nil {
		return sid, err
	}

	sid = s.Id
	return sid, err
}

// Events returns the event of the search specified by sid.
func (sc *SplunkClient) Events(sid string, offset int) (*Events, error) {

	eventURL := fmt.Sprintf("%s/services/search/jobs/%s/events", sc.baseURL, sid)

	data := url.Values{
		"offset":      {strconv.Itoa(offset)},
		"output_mode": {"json"},
	}

	e := &Events{}
	err := sc.send("GET", eventURL, data, e)
	if err != nil {
		return e, err
	}

	return e, nil
}

// StreamEvents returns a channel where events from a search specified by sid is streamed into
func (sc *SplunkClient) StreamEvents(sid string) <-chan interface{} {

	ch := make(chan interface{})
	var backoff, offset int
	go func() {
	outer:
		for !sc.stopped {
			e, err := sc.Events(sid, offset)
			if err != nil {
				ch <- err
				break outer
			}
			// Events didn't return any results. Sleep before trying again
			if len(e.Results) == 0 {
				// Break early on signal received
				select {
				case <-sc.stopCh:
					break outer
				case <-time.After(exp(backoff)):
				}
				backoff += 1
				continue
			}
			offset += len(e.Results)
			backoff = 0
			ch <- e
		}
		close(ch)
	}()

	return ch
}

// Shutdown stop any StreamEvents channels
func (sc *SplunkClient) Shutdown() {
	if !sc.stopped {
		close(sc.stopCh)
		sc.stopped = true
	}
}

// SetSessionKey sets the session key
func (sc *SplunkClient) SetSessionKey(key string) {
	sc.session.Key = key
}
