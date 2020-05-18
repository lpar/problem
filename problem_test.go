package problem_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/lpar/problem"
)

// Test that every recognized status code gives a unique type URL and title,
// and that the titles are title case -- except for HTTP 418 (see RFC 2324).
func TestForStatus(t *testing.T) {
	types := make(map[string]int)
	titles := make(map[string]int)
	for status := 400; status <= 599; status++ {
		prob := problem.New(status)
		if prob.Title != "" {
			if types[prob.Type] != 0 {
				t.Errorf("got type %s twice, for %d and %d", prob.Type, types[prob.Type], status)
			}
			if titles[prob.Title] != 0 {
				t.Errorf("got title %s twice, for %d and %d", prob.Title, titles[prob.Title], status)
			}
			types[prob.Type] = status
			titles[prob.Title] = status
			_, err := url.Parse(prob.Type)
			if err != nil {
				t.Errorf("got bad type URL for %d: %v", status, err)
			}
			if status != 418 {
				tct := strings.Title(prob.Title)
				if prob.Title != tct {
					t.Errorf("got %s expected %s", prob.Title, tct)
				}
			}
		}
	}
}

func TestNew(t *testing.T) {
	testdata := []struct{
		Status int
		Detail string
	}{
		{http.StatusNotFound, "No such page"},
		{http.StatusInsufficientStorage, "Server disk is full"},
	}
	for _, td := range testdata {
		testNew(t, td.Status, td.Detail)
	}
}

func testNew(t *testing.T, status int, msg string) {
	w := httptest.NewRecorder()
	prob := problem.New(status).Errorf(msg)
	problem.Write(w, prob)
	resp := w.Result()
	if resp.StatusCode != status {
		t.Errorf("got statuscode %d expected %d", resp.StatusCode, status)
	}
	st := strconv.Itoa(status) + " "
	if !strings.HasPrefix(resp.Status, st) {
		t.Errorf("got status %s expected start to be %s", resp.Status, st)
	}
	ct := resp.Header.Get("content-type")
	if ct != problem.ContentProblemDetails {
		t.Errorf("got content-type %s expected %s", ct, problem.ContentProblemDetails)
	}
	body,_ := ioutil.ReadAll(resp.Body)
	prob = &problem.ProblemDetails{}
 err := json.Unmarshal(body, prob)
	if err != nil {
		t.Error(err)
	}
	if prob.Status != status {
		t.Errorf("got statuscode %d in body expected %d", prob.Status, status)
	}
	if prob.Title == "" {
		t.Error("got blank title in body expected non-blank")
	}
	if prob.Type == "" {
		t.Error("got blank type in body expected non-blank")
	}
	if prob.Detail != msg {
		t.Errorf("got detail '%s' expected '%s'", prob.Detail, msg)
	}
	_, err = url.Parse(prob.Type)
	if err != nil {
		t.Errorf("bad type URL %s in body: %v", prob.Type, err)
	}
}

func roundTrip(t *testing.T, err error) problem.ProblemDetails {
	t.Helper()
	w := httptest.NewRecorder()
	problem.MustWrite(w, err)
	resp := w.Result()
	body,_ := ioutil.ReadAll(resp.Body)
	prob := problem.ProblemDetails{}
	err = json.Unmarshal(body, &prob)
	if err != nil {
		t.Error(err)
	}
	return prob
}

func TestReport(t *testing.T) {
	const errmsg1 = "This is not a problem report"
	err := errors.New(errmsg1)
	prob := roundTrip(t, err)
	if prob.Detail != errmsg1 {
		t.Errorf("expected '%s', got %s", errmsg1, prob.Detail)
	}
	const errmsg2 = "Page not found"
	err = problem.New(404).WithDetail(errmsg2)
	prob = roundTrip(t, err)
	if prob.Detail != errmsg2 {
		t.Errorf("expected '%s', got '%s'", errmsg2, prob.Detail)
	}
}