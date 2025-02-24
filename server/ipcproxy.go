package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/newtonproject/newchain-guard/params"
	log "github.com/sirupsen/logrus"
)

type IPCServer struct {
	rawURL    string
	bodyBytes []byte
	ErrorLog  *log.Logger
	LogReq    *params.LogRequest
}

func NewSingleIPCReverseProxy(rawURL string, bodyBytes []byte) *IPCServer {
	return &IPCServer{rawURL: rawURL, bodyBytes: bodyBytes}
}

func (s *IPCServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if cn, ok := w.(http.CloseNotifier); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		notifyChan := cn.CloseNotify()
		go func() {
			select {
			case <-notifyChan:
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	conn, err := net.Dial("unix", s.rawURL)
	if err != nil {
		params.LogAndResponseJSONError(w, r, s.ErrorLog, params.StatusIPCDialError, json.RawMessage{}, string(s.bodyBytes))
		return
	}
	defer conn.Close()

	_, err = conn.Write(s.bodyBytes)
	if err != nil {
		params.LogAndResponseJSONError(w, r, s.ErrorLog, params.StatusIPCWriteError, json.RawMessage{}, string(s.bodyBytes))
		return
	}

	var (
		buf    bytes.Buffer
		logBuf bytes.Buffer
	)
	size := 32 * 1024
	for {
		tmp := make([]byte, size) // using small tmo buffer for demonstrating
		n, err := conn.Read(tmp)
		if err != nil {
			params.LogAndResponseJSONError(w, r, s.ErrorLog, params.StatusIPCReadError, json.RawMessage{}, string(s.bodyBytes))
			return
		}
		buf.Write(tmp[:n])
		if n < size {
			// Force chunking if we saw a response trailer.
			// This prevents net/http from calculating the length for short
			// bodies and adding a Content-Length.
			logBuf.Write(buf.Bytes())
			w.Write(buf.Bytes())
			buf.Reset()
			if fl, ok := w.(http.Flusher); ok {
				fl.Flush()
			}
		}
		if n > 1 && tmp[n-1] == '\n' {
			break
		}
	}

	logBuf.Write(buf.Bytes())
	w.Write(buf.Bytes())

	if s.LogReq != nil {
		params.LogBatchRequestStructInfo(s.LogReq, logBuf.String())
	} else {
		params.LogRequestAndResponseInfo(r, s.ErrorLog, params.StatusOK, string(s.bodyBytes), strings.Trim(logBuf.String(), "\n"))
	}
}
