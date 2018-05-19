package sacn

var (
	activatedRecv = make(map[uint16]bool)
	// stores information about wether the universe is activated for receiving
)

func isActivated(universe uint16) bool {
	return activatedRecv[universe]
}
