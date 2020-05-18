package problem_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lpar/problem"
)

const msg1 = "Must be a valid e-mail address"
const msg2 = "You must provide your name"

func TestNewValidationProblem(t *testing.T) {
	valerr := problem.NewValidationProblem()
	valerr.Add("email", msg1)
	valerr.Add("name", msg2)
	w := httptest.NewRecorder()
	problem.MustWrite(w, valerr)
	resp := w.Result()
	body,_ := ioutil.ReadAll(resp.Body)
	prob := problem.ValidationProblem{}
	err := json.Unmarshal(body, &prob)
	if err != nil {
		t.Error(err)
	}
	if prob.Status != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, prob.Status)
	}
	if len(prob.ValidationErrors) != 2 {
		t.Errorf("got %d errors, expected 2", len(prob.ValidationErrors))
		return
	}
	p1 := prob.ValidationErrors[0]
	p2 := prob.ValidationErrors[1]
	var email problem.ValidationError
	var name problem.ValidationError
	if p1.FieldName == "email" {
		email = p1
		name = p2
	} else {
		email = p2
		name = p1
	}
	if email.FieldName != "email" || email.Error != msg1 {
		t.Errorf("lost/corrupted email field validation message")
	}
	if name.FieldName != "name" || name.Error != msg2 {
		t.Errorf("lost/corrupted name field validation message")
	}
}
