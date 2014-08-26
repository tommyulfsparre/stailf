package splunk

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

const MaxBackoff = 4 // 8s

// exp will return a time.Duration of 500 * 2**backoff
func exp(backoff int) time.Duration {

	if backoff >= MaxBackoff {
		backoff = MaxBackoff
	}

	return (500 * time.Millisecond *
		time.Duration(math.Exp2(float64(backoff))))
}

// genID will generate a unique search job id
func genID(params Param, suffix string) (id string, err error) {

	f, err := os.Open("/dev/urandom")
	if err != nil {
		return id, err
	}
	defer f.Close()

	b := make([]byte, 5)
	f.Read(b)

	var prefix string
	if v, ok := params["search_mode"]; ok {
		if v == "realtime" {
			prefix = "rt_"
		}
	}

	id = fmt.Sprintf("%s%x.%s", prefix, b, suffix)
	return id, err
}

// isStatusNotOk returns an error if the http response code is > 299
// http://docs.splunk.com/Documentation/Splunk/latest/RESTAPI/RESTintro#HTTP_status_codes
func isStatusOK(r *http.Response) error {
	if r.StatusCode > 299 {
		var m messages
		err := readJSON(r, &m)
		if err != nil {
			return err
		}

		if len(m.Messages) > 0 {
			return errors.New(m.Messages[0].Text)
		}
		return errors.New("No error messages returned from splunk")
	}

	return nil
}

// newInsecureHttpClient returns a new http.Client or if the scheme is https
// return a http.Client that doesn't validate the certificates.
func newInsecureHttpClient(url string) *http.Client {
	tr := &http.Transport{}
	if strings.HasPrefix(url, "https") {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
}

// readJSON will decode JSON from the response Body into v
func readJSON(r *http.Response, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&v)
	if err != nil {
		return err
	}

	return nil
}
