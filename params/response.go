package params

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/didip/tollbooth/libstring"
	log "github.com/sirupsen/logrus"
)

func LogAndResponseError(w http.ResponseWriter, r *http.Request, logger *log.Logger, status int, resBody []byte, body string) {
	LogRequestError(r, logger, status, body, string(resBody))
	w.Write(resBody)
}

func LogAndResponseJSONError(w http.ResponseWriter, r *http.Request, logger *log.Logger, status int, ID json.RawMessage, body string) {
	out := &JSONErrResponse{
		Version: JSONRPCVersion,
		ID:      ID,
		Error: JSONError{
			Code:    status,
			Message: fmt.Sprintf("%s - %d", ErrorInternalError.Error(), status),
		},
	}
	buf, err := json.Marshal(out)
	if err != nil {
		LogAndResponseError(w, r, logger, status, nil, body)
		return
	}
	buf = append(buf, '\n')
	w.Header().Set("Content-Type", "application/json")
	LogAndResponseError(w, r, logger, status, buf, body)
}

func LogAndResponseOK(w http.ResponseWriter, r *http.Request, logger *log.Logger, status int, resBody []byte, reqBody string) {
	LogRequestAndResponseInfo(r, logger, status, reqBody, string(resBody))
	w.Write(resBody)
}

func LogRequestInfo(r *http.Request, logger *log.Logger, status int, body string) {
	if r != nil {
		xForwardedFor := r.Header.Get("X-Forwarded-For")
		remoteIP := getRemoteIP(r)
		request := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)
		server := r.Host

		if logger != nil {
			go func() {
				fields := log.Fields{
					"status":  status,
					"client":  remoteIP,
					"server":  server,
					"request": request,
				}
				if xForwardedFor != "" {
					fields["X-Forwarded-For"] = xForwardedFor
				}
				logger.WithFields(fields).Info(body)
			}()
		}
	}
}

func LogRequestAndResponseInfo(r *http.Request, logger *log.Logger, status int, body, resBody string) {
	if r != nil {
		xForwardedFor := r.Header.Get("X-Forwarded-For")
		remoteIP := getRemoteIP(r)
		request := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)
		server := r.Host

		if logger != nil {
			go func() {
				fields := log.Fields{
					"status":   status,
					"client":   remoteIP,
					"server":   server,
					"request":  request,
					"response": resBody,
				}
				if xForwardedFor != "" {
					fields["X-Forwarded-For"] = xForwardedFor
				}
				logger.WithFields(fields).Info(body)
			}()
		}
	}
}

func LogRequestError(r *http.Request, logger *log.Logger, status int, body string, resBody string) {
	if r != nil {
		xForwardedFor := r.Header.Get("X-Forwarded-For")
		remoteIP := getRemoteIP(r)
		request := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)
		server := r.Host

		if logger != nil {
			go func() {
				fields := log.Fields{
					"status":   status,
					"client":   remoteIP,
					"server":   server,
					"request":  request,
					"response": resBody,
				}
				if xForwardedFor != "" {
					fields["X-Forwarded-For"] = xForwardedFor
				}
				logger.WithFields(fields).Error(body)
			}()
		}
	}
}

func LogRequestWarn(r *http.Request, logger *log.Logger, status int, body string) {
	if r != nil {
		xForwardedFor := r.Header.Get("X-Forwarded-For")
		remoteIP := getRemoteIP(r)
		request := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)
		server := r.Host

		if logger != nil {
			go func() {
				fields := log.Fields{
					"status":  status,
					"client":  remoteIP,
					"server":  server,
					"request": request,
				}
				if xForwardedFor != "" {
					fields["X-Forwarded-For"] = xForwardedFor
				}
				logger.WithFields(fields).Warn(body)
			}()
		}
	}
}

func getRemoteIP(r *http.Request) string {
	return libstring.RemoteIP([]string{"X-Forwarded-For", "RemoteAddr", "X-Real-IP"}, 0, r)
}

func LogBatchRequestAndResponse(w http.ResponseWriter, r *http.Request, logger *log.Logger, statusList []int, reqBody string, resBody []byte) {
	LogBatchRequestInfo(r, logger, statusList, reqBody, string(resBody))
	w.Write(resBody)
}

func LogBatchRequestStructInfo(lr *LogRequest, resBody string) {
	if !lr.IsBatch && len(lr.StatusList) > 0 {
		LogRequestAndResponseInfo(lr.R, lr.Logger, lr.StatusList[0], string(lr.ReqBody), resBody)
	} else {
		LogBatchRequestInfo(lr.R, lr.Logger, lr.StatusList, string(lr.ReqBody), resBody)
	}
}

func LogBatchRequestInfo(r *http.Request, logger *log.Logger, statusList []int, reqBody, resBody string) {
	if r != nil {
		xForwardedFor := r.Header.Get("X-Forwarded-For")
		remoteIP := getRemoteIP(r)
		request := fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto)
		server := r.Host

		if logger != nil {
			go func() {
				fields := log.Fields{
					"status":  strings.Replace(fmt.Sprint(statusList), " ", ",", -1),
					"client":  remoteIP,
					"server":  server,
					"request": request,
				}
				if resBody != "" {
					fields["response"] = resBody
				}
				if xForwardedFor != "" {
					fields["X-Forwarded-For"] = xForwardedFor
				}
				logger.WithFields(fields).Info(reqBody)
			}()
		}
	}
}

type LogRequest struct {
	R          *http.Request
	Logger     *log.Logger
	StatusList []int
	ReqBody    []byte
	IsBatch    bool
}
