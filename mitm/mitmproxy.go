package mitm

import (
	"log"
	"net/http"
	"os"
	"time"

	"pub.evening/mitm/v2/certs"
	"pub.evening/mitm/v2/config"
)

var logger *log.Logger

// Gomitmproxy create a mitm proxy and start it
func Gomitmproxy(conf *config.Cfg, ch chan bool) {
	certs, err := certs.New(&certs.Options{
		CacheSize: 256,
		Directory: ".",
	})
	if err != nil {
		log.Fatal(err)
	}
	tlsConfig := certs.TLSConfigFromCA()
	handler := InitConfig(certs, tlsConfig)
	server := &http.Server{
		Addr:         ":" + *conf.Port,
		ReadTimeout:  1 * time.Hour,
		WriteTimeout: 1 * time.Hour,
		Handler:      handler,
	}

	l, _ := os.Create(*conf.Log)
	logger = log.New(l, "[mitmproxy]", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	logger.Println("Server is listening at ", server.Addr)

	go func() {
		server.ListenAndServe()
		ch <- true
	}()

	return
}
