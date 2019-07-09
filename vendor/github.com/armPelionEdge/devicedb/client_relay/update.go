package client_relay

type Update struct {
	Key string
	Serial uint64
	Context string
	Siblings []string
	LastStableSerial uint64
}

func (update *Update) IsEmpty() bool {
	return update.Key == ""
}