module gitlab.newtonproject.org/yangchenzhong/NewChainGuard

go 1.15

require (
	github.com/allegro/bigcache v1.2.1 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/didip/tollbooth v4.0.2+incompatible
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/ethereum/go-ethereum v1.10.15
	github.com/go-kit/kit v0.9.0 // indirect
	github.com/go-sql-driver/mysql v1.5.0
	github.com/junhsieh/goexamples v0.0.0-20190721045834-1c67ae74caa6
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/peterh/liner v1.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/upper/db/v4 v4.0.0
	github.com/yuin/gopher-lua v0.0.0-20200816102855-ee81675732da
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
)

replace github.com/ethereum/go-ethereum => github.com/newtonproject/newchain v1.10.15-newton
