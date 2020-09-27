package logging

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
)

// JSONLogFormatter formats the output for Logrus JSON output
type JSONLogFormatter struct {
	ServerAddress *net.TCPAddr
}

// JSONLogEntry prepares the Logrus context
type JSONLogEntry struct {
	request       *http.Request
	serverAddress *net.TCPAddr
}

// NewLogEntry creates a new entry for the Logrus log
func (j *JSONLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &JSONLogEntry{
		request:       r,
		serverAddress: j.ServerAddress,
	}
}

func getHeader(headers http.Header, name string) string {
	if name == "" {
		return ""
	}
	name = http.CanonicalHeaderKey(name)

	found, ok := headers[name]
	if ok {
		if len(found) > 0 {
			return found[0]
		}
		return ""
	}
	return ""
}

// Write outputs the log entry into the log
func (j *JSONLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {

	r := j.request

	remoteUser, _, basicAuthOk := r.BasicAuth()
	if !basicAuthOk {
		remoteUser = ""
	}

	logrus.
		WithFields(logrus.Fields{
			"vhost":                   r.Host,
			"hostname":                r.Host,
			"remote_addr":             r.RemoteAddr,
			"remote_user":             remoteUser,
			"x-forwarded-for":         getHeader(r.Header, "X-Forwarded-For"),
			"request":                 fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto),
			"request_time":            elapsed.Seconds(),
			"request_length":          r.ContentLength,
			"request_id":              middleware.GetReqID(r.Context()),
			"request_method":          r.Method,
			"request_completion":      "OK",
			"request_uri":             r.RequestURI,
			"query_string":            r.URL.RawQuery,
			"server_protocol":         r.Proto,
			"server_port":             j.serverAddress.Port,
			"status":                  status,
			"received_referrer":       r.Referer(),
			"received_content_length": r.ContentLength,
			"received_content_type":   getHeader(r.Header, "Content-Type"),
			"received_cookie":         getHeader(r.Header, "Cookie"),
			"sent_bytes":              bytes,
			"sent_content_type":       getHeader(header, "Content-Type"),
			"sent_content_range":      getHeader(header, "Content-Range"),
			"sent_etag":               getHeader(header, "Etag"),
			"sent_last_modified":      getHeader(header, "Last-Modified"),
			"protocol":                "HTTP",
			"app":                     "ticker",
			"type":                    "access",
			"user_agent":              r.UserAgent(),
			"extra":                   extra,
		}).
		Debug()
}

// Panic outputs the log entry into the log
func (j *JSONLogEntry) Panic(v interface{}, stack []byte) {
	r := j.request

	remoteUser, _, basicAuthOk := r.BasicAuth()
	if !basicAuthOk {
		remoteUser = ""
	}

	logrus.
		WithFields(logrus.Fields{
			"vhost":                   r.Host,
			"hostname":                r.Host,
			"remote_addr":             r.RemoteAddr,
			"remote_user":             remoteUser,
			"x-forwarded-for":         getHeader(r.Header, "X-Forwarded-For"),
			"request":                 fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto),
			"request_length":          r.ContentLength,
			"request_id":              middleware.GetReqID(r.Context()),
			"request_method":          r.Method,
			"request_completion":      "",
			"request_uri":             r.RequestURI,
			"query_string":            r.URL.RawQuery,
			"server_protocol":         r.Proto,
			"server_port":             j.serverAddress.Port,
			"received_referrer":       r.Referer(),
			"received_content_length": r.ContentLength,
			"received_content_type":   getHeader(r.Header, "Content-Type"),
			"received_cookie":         getHeader(r.Header, "Cookie"),
			"protocol":                "HTTP",
			"app":                     "ticker",
			"type":                    "access",
			"user_agent":              r.UserAgent(),
			"error":                   v,
			"stack":                   string(stack),
		}).
		Errorf("%+v", v)
}
