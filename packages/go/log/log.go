package log

import "go.uber.org/zap"

const (
	ServiceRoutineKey = "service_routine"
	CacheIdKey = "cache_id"
	URLKey = "url"
	NumberOfWaitingClientsKey = "num_waiting_clients"
)

func FServiceRoutine(name string) zap.Field {
	return zap.String(ServiceRoutineKey, name)
}

func FCacheId(name string) zap.Field {
	return zap.String(CacheIdKey, name)
}

func FURL(name string) zap.Field {
	return zap.String(URLKey, name)
}

func FNumberOfWaitingClients(num int) zap.Field {
	return zap.Int(NumberOfWaitingClientsKey, num)
}
