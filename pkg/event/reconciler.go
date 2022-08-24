package event

// Reconciler updates list of available listeners in the background as we don't want to load them on each event
type Reconciler struct {
	Reconcilers []ListenerReconiler
}

// Register registers new listener reconciler
func (s *Reconciler) Register(reconciler ListenerReconiler) {
	s.Reconcilers = append(s.Reconcilers, reconciler)
}

func (s *Reconciler) Reconcile() (listeners []Listener) {
	listeners = make([]Listener, 0)
	for _, reconciler := range s.Reconcilers {
		listeners = append(listeners, reconciler.Load()...)
	}

	return listeners
}
