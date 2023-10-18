package logging

type Config struct {
	Type string

	Host  string
	Port  uint
	Token string

	Organization string
	Bucket       string
}
