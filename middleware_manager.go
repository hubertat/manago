package manago

type Middleware interface {
	RunBefore(*Controller) bool
	RunAfter(*Controller)
}
