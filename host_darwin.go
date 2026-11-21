package main

// toolWrap (macOS): the tool with its box's arguments (sandbox.go, host.go's
// boxFor); the environment passes through, as to any program started by hand.
func (h *Host) toolWrap(name, bin string, args []string) (string, []string, []string, error) {
	run, argv := h.boxFor(name, bin).argv(args...)
	return run, argv, nil, nil
}
