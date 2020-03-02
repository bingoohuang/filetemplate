package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bingoohuang/filetemplate"
	"github.com/sirupsen/logrus"
)

func main() {
	http.HandleFunc("/file", wrap(file))
}

type serveHandler func(w http.ResponseWriter, r *http.Request)

type resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   interface{} `json:"error"`
}

type clientError struct {
	Code    int
	Message string
	Err     error
	Data    interface{}
}

func (c *clientError) Error() string {
	if c.Message != "" {
		if c.Err != nil {
			return fmt.Sprintf(c.Message, c.Err)
		}

		return c.Message
	}

	if c.Err != nil {
		return c.Err.Error()
	}

	return "client error occurred"
}

func (c *clientError) GetCode() int {
	if c.Code == 0 {
		return http.StatusBadRequest
	}

	return c.Code
}

func wrap(f func(w http.ResponseWriter, r *http.Request) error) serveHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			_ = writeError(w, err)
		} else {
			_ = writeJSON(w, resp{Code: 0, Message: "OK"})
		}
	}
}

func writeError(w http.ResponseWriter, err error) error {
	var e *clientError
	if errors.As(err, &e) {
		return writeJSON(w, resp{Code: e.GetCode(), Message: e.Message, Data: e.Data})
	}

	return writeJSON(w, err.Error())
}

func writeJSON(w http.ResponseWriter, v interface{}) error {
	_, ok1 := v.(resp)
	_, ok2 := v.(*resp)

	if !ok1 && !ok2 {
		v = resp{Code: 0, Message: "ok", Data: v}
	}

	if body, err := json.Marshal(v); err != nil {
		logrus.Errorf("failed to json.Marshal %+v", v)
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(err.Error()))
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(body)
	}

	return nil
}

func file(w http.ResponseWriter, r *http.Request) error {
	body := r.Body
	defer body.Close()

	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return &clientError{Message: "fail to ioutil.ReadAll %v", Err: err}
	}

	f := &filetemplate.File{}

	err = json.Unmarshal(bodyBytes, f)
	if err != nil {
		return &clientError{Message: "fail to json.Unmarshal %v", Err: err}
	}

	return f.Execute()
}
