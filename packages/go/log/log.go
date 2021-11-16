package log

import "go.uber.org/zap"

const (
	ServiceRoutineKey         = "service_routine"
	CacheIDKey                = "cache_id"
	URLKey                    = "url"
	NumberOfWaitingClientsKey = "num_waiting_clients"
)

func FServiceRoutine(name string) zap.Field {
	return zap.String(ServiceRoutineKey, name)
}

func FCacheID(name string) zap.Field {
	return zap.String(CacheIDKey, name)
}

func FURL(name string) zap.Field {
	return zap.String(URLKey, name)
}

func FNumberOfWaitingClients(num int) zap.Field {
	return zap.Int(NumberOfWaitingClientsKey, num)
}
