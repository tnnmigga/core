package conf

import "log"

var (
	ServerID   uint32
	ServerType string
)

func initServerConf() {
	ServerID = UInt32("server.id")
	ServerType = String("server.type")
	if len(ServerType) == 0 {
		log.Fatal("server.type is empty")
	}
}
