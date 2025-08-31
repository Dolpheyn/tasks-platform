package taskmanager

type PickupSignal struct {
	ClientID string
}

type DropSignal struct {
	Reason string
}

type ResultSignal struct {
	Success bool
	Msg     string
}
