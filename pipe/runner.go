package pipe

type Runner struct {
	startNodes    []startable
	terminalNodes []doneable
}
