package notify

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

var (
	ClientID = "guard"
	Topic    = "RawTransaction"
	QoS      = byte(1)
)

var (
	TxChanMax = 4096
	txCh      chan string
	Logger    *log.Logger
)

type warnLogger struct {
	*log.Logger
}

func (e warnLogger) Println(v ...interface{}) {
	e.Warnln(v)
}

func (e warnLogger) Printf(format string, v ...interface{}) {
	e.Warnf(format, v...)
}

func InitAndRunNotifyClient(server, username, password string, logger *log.Logger) {
	if logger == nil {
		logger = log.New()
	}
	Logger = logger
	c, err := newClient(server, username, password)
	if err != nil {
		logger.Warn(err)
		return
	}

	txCh = make(chan string, TxChanMax)

	Run(c)
}

func Run(client mqtt.Client) {
	for txRaw := range txCh {
		if client != nil {
			client.Publish(Topic, QoS, false, txRaw)
		}
	}
}

func newClient(server, username, password string) (mqtt.Client, error) {
	if Logger != nil {
		mqtt.ERROR = warnLogger{Logger}
	}
	opts := mqtt.NewClientOptions().AddBroker(server).SetClientID(ClientID)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.OnConnect = func(c mqtt.Client) {
		Logger.Info("ActiveMQ Connected/Reconnected...")
	}

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return c, nil
}

func AddTx(raw string) {
	select {
	case txCh <- raw:
	default:
		Logger.Warnln("discard raw transaction ", raw)
	}
}
