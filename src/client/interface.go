package client

type RateLimiter interface {
	Aquire(key string, maxConncurrency, maxVolume int) (int, error)
	Return(key string, volume int) (int, error)
}
