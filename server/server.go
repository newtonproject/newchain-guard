package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"

	"github.com/newtonproject/newchain-guard/filter"
	"github.com/newtonproject/newchain-guard/params"
	"github.com/sirupsen/logrus"
)

type Server struct {
	config *params.Config
	// mu sync.RWMutex
	ErrorLog *logrus.Logger
}

func NewServer(config *params.Config) *Server {
	return &Server{config: config}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Permit dumb empty requests for remote health-checks (AWS)
	if r.Method == http.MethodGet && r.ContentLength == 0 && r.URL.RawQuery == "" {
		return
	}
	if code, err := validateRequest(r); err != nil {
		http.Error(w, err.Error(), code)
		return
	}

	// s.mu.Lock()
	// config := params.Copy(s.config)
	// s.mu.Unlock()
	// config := params.Copy(s.config)
	config := s.config

	u, err := url.Parse(config.RawURL)
	if err != nil {
		fmt.Println(err)
		return
	}

	if r.Body == nil {
		params.LogAndResponseJSONError(w, r, s.ErrorLog, params.StatusBodyNilOrEmpty, json.RawMessage{}, "")
		return
	}
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		params.LogAndResponseJSONError(w, r, s.ErrorLog, params.StatusReadBodyError, json.RawMessage{}, "")
		return
	}

	var logReq *params.LogRequest
	if r.Method != http.MethodOptions {
		f, err := filter.NewFilter(config, s.ErrorLog)
		if err != nil {
			params.LogAndResponseJSONError(w, r, s.ErrorLog, params.StatusFilterNoConfig, json.RawMessage{}, string(bodyBytes))
			return
		}
		// if err := f.CheckJSONRequest(w, r, bodyBytes); err != nil {
		// 	return
		// }
		logReq, err = f.HandleJSONRequest(w, r, bodyBytes)
		if err == params.ErrorGuard {
			// just return
			return
		} else if err != nil {
			logrus.WithField("err1", err).Errorln(string(bodyBytes))
			params.LogAndResponseJSONError(w, r, s.ErrorLog, params.StatusInternalError, json.RawMessage{}, string(bodyBytes))
			return
		}
	}

	switch u.Scheme {
	case "http", "https":
		body := bytes.NewReader(bodyBytes)
		rc, ok := io.Reader(body).(io.ReadCloser)
		if !ok && body != nil {
			rc = ioutil.NopCloser(body)
		}
		r.Body = rc

		p := NewSingleHostReverseProxy(u)
		p.ErrorLog = s.ErrorLog
		p.LogReq = logReq
		if p.LogReq == nil {
			p.LogReq = &params.LogRequest{
				R:          r,
				Logger:     p.ErrorLog,
				StatusList: []int{params.StatusOK},
				ReqBody:    bodyBytes,
				IsBatch:    false,
			}
		}
		p.ServeHTTP(w, r)

		return
	case "":
		proxy := NewSingleIPCReverseProxy(config.RawURL, bodyBytes)
		proxy.ErrorLog = s.ErrorLog
		proxy.LogReq = logReq
		proxy.ServeHTTP(w, r)
		return

	}

	return
}

// validateRequest returns a non-zero response code and error message if the
// request is invalid.
func validateRequest(r *http.Request) (int, error) {
	if r.Method == http.MethodPut || r.Method == http.MethodDelete {
		return http.StatusMethodNotAllowed, errors.New("method not allowed")
	}
	if r.ContentLength > params.MaxRequestContentLength {
		err := fmt.Errorf("content length too large (%d>%d)", r.ContentLength, params.MaxRequestContentLength)
		return http.StatusRequestEntityTooLarge, err
	}
	mt, _, err := mime.ParseMediaType(r.Header.Get("content-type"))
	if r.Method != http.MethodOptions && (err != nil || mt != params.ContentType) {
		err := fmt.Errorf("invalid content type, only %s is supported", params.ContentType)
		return http.StatusUnsupportedMediaType, err
	}
	return 0, nil
}
