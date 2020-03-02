package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bingoohuang/filetemplate"
	flags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

// App defines the current app
type App struct {
	Addr string `short:"a" long:"addr" description:"bind address" default:":3003"`
}

func main() {
	app := &App{}

	_, err := flags.Parse(app)
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			os.Exit(0)
		}

		os.Exit(1) // nolint gomnd
	}

	logrus.Infof("app started at %s", app.Addr)

	http.HandleFunc("/file", wrap(file))

	if err := http.ListenAndServe(app.Addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "ListenAndServe error %v", err)
		os.Exit(1) // nolint gomnd
	}
}

type serveHandler func(w http.ResponseWriter, r *http.Request)

type resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
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

func wrap(f func(w http.ResponseWriter, r *http.Request) (interface{}, error)) serveHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		if v, err := f(w, r); err != nil {
			_ = writeError(w, err)
		} else {
			_ = writeJSON(w, resp{Code: 0, Message: "OK", Data: v})
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

func file(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	body := r.Body
	defer body.Close()

	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, &clientError{Message: "fail to ioutil.ReadAll %v", Err: err}
	}

	f := &filetemplate.File{}

	err = json.Unmarshal(bodyBytes, f)
	if err != nil {
		return nil, &clientError{Message: "fail to json.Unmarshal %v", Err: err}
	}

	return f.Execute()
}
