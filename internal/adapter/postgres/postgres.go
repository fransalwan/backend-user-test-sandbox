package postgres

// Config holds connection parameters for PostgreSQL.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}
