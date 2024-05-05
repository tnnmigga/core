package msgbus

import (
	"time"

	"github.com/tnnmigga/core/conf"
	"github.com/tnnmigga/core/idef"
	"github.com/tnnmigga/core/utils"
)

type castOpt struct {
	key   string
	value any
}

func findCastOpt[T any](opts []castOpt, key string, defaultVal T) (value T) {
	for _, opt := range opts {
		if opt.key == key {
			return opt.value.(T)
		}
	}
	return defaultVal
}

func UseStream() castOpt {
	return castOpt{
		key:   idef.ConstKeyUseStream,
		value: true,
	}
}

func OneOfMods(modName idef.ModName) castOpt {
	return castOpt{
		key:   idef.ConstKeyOneOfMods,
		value: modName,
	}
}

func ServerID(serverID uint32) castOpt {
	return castOpt{
		key:   idef.ConstKeyServerID,
		value: serverID,
	}
}

func ServerType(serverType string) castOpt {
	return castOpt{
		key:   idef.ConstKeyServerType,
		value: serverType,
	}
}

func Expires(expires time.Duration) castOpt {
	return castOpt{
		key:   idef.ConstKeyExpires,
		value: int64(utils.NowNs() + expires),
	}
}

func Local() castOpt {
	return castOpt{
		key:   idef.ConstKeyServerID,
		value: conf.ServerID,
	}
}
