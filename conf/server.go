package conf

import (
	"fmt"
)

var (
	ServerID   uint32
	ServerType string
)

func ckeckServer() error {
	ServerID = Uint32("server.id")
	if ServerID <= 0 || ServerID >= 4096 {
		return fmt.Errorf("server id error %d", ServerID)
	}
	ServerType = String("server.type")
	if len(ServerType) == 0 {
		return fmt.Errorf("server.type is empty")
	}
	return nil
}
