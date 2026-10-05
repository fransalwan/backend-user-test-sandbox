package redis

// Config holds connection parameters for Redis.
type Config struct {
	Addr     string
	Password string
	DB       int
}
