package systemgate

import "sync/atomic"

type Gate struct{ enabled atomic.Bool }

func (g *Gate) Enable()       { g.enabled.Store(true) }
func (g *Gate) Disable()      { g.enabled.Store(false) }
func (g *Gate) Enabled() bool { return g.enabled.Load() }
