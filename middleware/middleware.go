package middleware

type Controlled interface {
	SetError(httpErrorCode int, err error, errorString ...string)
	GetRedir() (isSet bool, path string)
	SetRedir(path string)
}

type Middleware interface {
	RunBefore(Controlled, map[string]string) bool
	RunAfter(Controlled, map[string]string)
}
