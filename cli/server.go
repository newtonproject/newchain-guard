package cli

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.newtonproject.org/yangchenzhong/NewChainGuard/notify"
	"gitlab.newtonproject.org/yangchenzhong/NewChainGuard/server"
)

func (cli *CLI) buildServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "server",
		Short:                 "Run as reverse proxy server",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			sslCertificate := viper.GetString("SSLCertificate")
			sslCertificateKey := viper.GetString("SSLCertificateKey")

			port := viper.GetInt("port")
			host := viper.GetString("host")
			hostAddress := fmt.Sprintf("%s:%d", host, port)

			logger := logrus.New()
			logger.Out = os.Stdout
			logLevel := viper.GetString("LogLevel")
			if logLevel == "" {
				logLevel = "info"
			}
			level, err := logrus.ParseLevel(logLevel)
			if err != nil {
				logger.Error(err)
				return
			}
			logger.SetLevel(level)

			config, err := loadParamsConfig()
			if err != nil {
				logger.Error(err)
				return
			}
			if config == nil {
				logger.Errorln("config is nil")
				return
			}
			b, err := json.MarshalIndent(config, "", "\t")
			if err != nil {
				fmt.Println(err)
				return
			}
			logger.Printf("The config is as follow: \n%s\n", string(b))

			if config.EnableActiveMQ {
				if clientID := viper.GetString("ActiveMQ.ClientID"); clientID != "" {
					notify.ClientID = clientID
				}
				if viper.IsSet("ActiveMQ.QoS") {
					if qos := viper.GetInt("ActiveMQ.QoS"); qos == 0 || qos == 1 || qos == 2 {
						notify.QoS = byte(qos)
					}
				} else {
					notify.QoS = byte(1)
				}
				if topic := viper.GetString("ActiveMQ.Topic"); topic != "" {
					notify.Topic = topic
				}

				go func() {
					notify.InitAndRunNotifyClient(
						viper.GetString("ActiveMQ.Server"),
						viper.GetString("ActiveMQ.Username"),
						viper.GetString("ActiveMQ.Password"),
						logger)
				}()
			}

			s := server.NewServer(config)
			s.ErrorLog = logger
			if viper.GetBool("EnableIPRateLimit") {
				IPRate := viper.GetFloat64("IPRate")
				if IPRate <= 0 {
					IPRate = 1000
				}
				logger.Printf("The IP Rate Limit, which is limited to %d request per second, is Enabled", int64(IPRate))
				lmt := tollbooth.NewLimiter(IPRate, &limiter.ExpirableOptions{
					DefaultExpirationTTL: time.Second * 30})
				lmt.SetIPLookups([]string{"X-Forwarded-For", "RemoteAddr", "X-Real-IP"})
				lmt.SetMethods([]string{"POST"})
				http.Handle("/", tollbooth.LimitFuncHandler(lmt, s.ServeHTTP))
			} else {
				http.HandleFunc("/", s.ServeHTTP)
			}

			escrow := func(w http.ResponseWriter, r *http.Request) {
				fmt.Println("escrow")
				u, err := url.Parse(viper.GetString("escrow"))
				if err != nil {
					fmt.Println(err)
					return
				}
				proxy := httputil.NewSingleHostReverseProxy(u)
				proxy.ServeHTTP(w, r)
			}
			http.HandleFunc("/escrow.Escrow", escrow)
			http.HandleFunc("/escrow", escrow)

			httpRouters := viper.GetStringMapString("HTTPRouters")
			handleRouter := func(p, u string) {
					target, err := url.Parse(u)
					if err != nil {
						logger.Error(err)
						return
					}
					proxyFunc := func(w http.ResponseWriter, r *http.Request) {
						proxy := server.NewSingleHostReverseProxy2(target, p)
						proxy.ErrorLog = logger
						proxy.ServeHTTP(w, r)
					}
					p = "/" + p
					http.HandleFunc(p, proxyFunc)
					logger.Printf("Proxy '%s' to '%s'", p, target.String())
			}
			if len(httpRouters) > 0 {
				for p, u := range httpRouters {
					handleRouter(p,u)
				}
			}

			// httpListenAndServe.(hostAddress, nil)
			srv := &http.Server{
				Addr:         hostAddress,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
				IdleTimeout:  120 * time.Second,
				TLSConfig: &tls.Config{
					PreferServerCipherSuites: true,
					CurvePreferences:         []tls.CurveID{tls.CurveP256},
					CipherSuites:             []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
				},
				ErrorLog: log.New(logger.Writer(), "Guaid", 0),
			}

			if sslCertificate != "" && sslCertificateKey != "" {
				logger.Printf("Listening on %s with ssl enabled", srv.Addr)
				err = srv.ListenAndServeTLS(sslCertificate, sslCertificateKey)
			} else {
				logger.Printf("Listening on %s", srv.Addr)
				err = srv.ListenAndServe()
			}
			if err != nil {
				logger.Error(err)
				return
			}
		},
	}

	cmd.Flags().IntP("port", "p", 80, "the `port` of server")
	cmd.Flags().StringP("host", "H", "0.0.0.0", "the `host` of server")
	viper.BindPFlag("port", cmd.Flags().Lookup("port"))
	viper.BindPFlag("host", cmd.Flags().Lookup("host"))

	return cmd
}
