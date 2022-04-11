package mitm

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"time"

	"pub.evening/mitm/v2/certs"
)

const (
	Version   = "1.1"
	ONE_DAY   = 24 * time.Hour
	TWO_WEEKS = ONE_DAY * 14
	ONE_MONTH = 1
	ONE_YEAR  = 1
)

// HandlerWrapper wrapper of handler for http server
type HandlerWrapper struct {
	Certs     *certs.Manager
	wrapped   http.Handler
	tlsConfig func(host string) (*tls.Config, error)
	https     bool
}

// InitConfig init HandlerWrapper
func InitConfig(certs *certs.Manager, tlsconfig func(host string) (*tls.Config, error)) *HandlerWrapper {
	handler := &HandlerWrapper{
		Certs:     certs,
		tlsConfig: tlsconfig,
	}
	return handler
}

// ServeHTTP the main function interface for http handler
func (handler *HandlerWrapper) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method == "CONNECT" {
		handler.https = true
		handler.InterceptHTTPS(resp, req)
	} else {
		handler.https = false
		handler.DumpHTTPAndHTTPS(resp, req)
	}
}

// DumpHTTPAndHTTPS function to dump the HTTP request header and body
func (handler *HandlerWrapper) DumpHTTPAndHTTPS(resp http.ResponseWriter, req *http.Request) {
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Accept-Encoding")
	req.Header.Set("Connection", "Keep-Alive")

	var reqDump []byte
	ch := make(chan bool)
	// handle connection
	go func() {
		reqDump, _ = httputil.DumpRequestOut(req, true)
		ch <- true
	}()

	connHj, _, err := resp.(http.Hijacker).Hijack()
	if err != nil {
		logger.Println("Hijack fail to take over the TCP connection from client's request")
	}
	defer connHj.Close()

	host := req.Host

	matched, _ := regexp.MatchString(":[0-9]+$", host)

	var connOut net.Conn
	if !handler.https {
		if !matched {
			host += ":80"
		}
		connOut, err = net.DialTimeout("tcp", host, time.Second*30)
		if err != nil {
			logger.Println("Dial to", host, "error:", err)
			return
		}
	} else {
		if !matched {
			host += ":443"
		}
		connOut, err = tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			logger.Println("Dial to", host, "error:", err)
			return
		}
	}

	// Write writes an HTTP/1.1 request, which is the header and body, in wire format. This method consults the following fields of the request:
	/*
		Host
		URL
		Method (defaults to "GET")
		Header
		ContentLength
		TransferEncoding
		Body
	*/
	if err = req.Write(connOut); err != nil {
		logger.Println("send to server error", err)
		return
	}

	respFromRemote, err := http.ReadResponse(bufio.NewReader(connOut), req)
	if err != nil && err != io.EOF {
		logger.Println("Fail to read response from remote server.", err)
	}

	respDump, err := httputil.DumpResponse(respFromRemote, true)
	if err != nil {
		logger.Println("Fail to dump the response.", err)
	}
	// Send remote response back to client
	_, err = connHj.Write(respDump)
	if err != nil {
		logger.Println("Fail to send response back to client.", err)
	}

	<-ch
	// why write to reqDump, and in httpDump resemble to req again
	// in test, i find that the req may be destroyed by sth i currently dont know
	// so while parsing req in httpDump directly, it will raise execption
	// so dump its content to reqDump first.
	go httpDump(reqDump, respFromRemote)
}

// InterceptHTTPS to dump data in HTTPS
func (handler *HandlerWrapper) InterceptHTTPS(resp http.ResponseWriter, req *http.Request) {
	log.Println("An HTTPS GOT!")
	addr := req.Host
	host := strings.Split(addr, ":")[0]

	tlsConfig, err := handler.tlsConfig(host)
	if err != nil {
		logger.Printf("Could not get mitm cert for name: %s\nerror: %s\n", host, err)
		respBadGateway(resp)
		return
	}

	connIn, _, err := resp.(http.Hijacker).Hijack()
	if err != nil {
		logger.Printf("Unable to access underlying connection from client: %s\n", err)
		respBadGateway(resp)
		return
	}

	tlsConnIn := tls.Server(connIn, tlsConfig)
	listener := &mitmListener{tlsConnIn}
	httpshandler := http.HandlerFunc(func(resp2 http.ResponseWriter, req2 *http.Request) {
		req2.URL.Scheme = "https"
		req2.URL.Host = req2.Host
		handler.DumpHTTPAndHTTPS(resp2, req2)
	})

	go func() {
		err = http.Serve(listener, httpshandler)
		if err != nil && err != io.EOF {
			logger.Printf("Error serving mitm'ed connection: %s", err)
		}
	}()

	connIn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
}

func copyHTTPRequest(template *http.Request) *http.Request {
	req := &http.Request{}
	if template != nil {
		*req = *template
	}
	return req
}

func respBadGateway(resp http.ResponseWriter) {
	resp.WriteHeader(502)
}
