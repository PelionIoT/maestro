package dhcp4

type OptionCode byte

type Option struct {
	Code  OptionCode
	Value []byte
}

// Map of DHCP options
type Options map[OptionCode][]byte

// SelectOrderOrAll has same functionality as SelectOrder, except if the order
// param is nil, whereby all options are added (in arbitrary order).
func (o Options) SelectOrderOrAll(order []byte) []Option {
	if order == nil {
		opts := make([]Option, 0, len(o))
		for i, v := range o {
			opts = append(opts, Option{Code: i, Value: v})
		}
		return opts
	}
	return o.SelectOrder(order)
}

// SelectOrder returns a slice of options ordered and selected by a byte array
// usually defined by OptionParameterRequestList.  This result is expected to be
// used in ReplyPacket()'s []Option parameter.
func (o Options) SelectOrder(order []byte) []Option {
	opts := make([]Option, 0, len(order))
	for _, v := range order {
		if data, ok := o[OptionCode(v)]; ok {
			opts = append(opts, Option{Code: OptionCode(v), Value: data})
		}
	}
	return opts
}

type OpCode byte
type MessageType byte // Option 53

// MakeClientIdentifier is for use with Option 61 OptionClientIdentifier
func MakeClientIdentifier(clientType byte, addr []byte) (ret []byte) {
	// typically the MAC is 6 bytes, so 7 would be the max. But it could be 9 if using 802.15.4 or some
	// future protocol with a 64-bit MAC
	ret = make([]byte, 0, 9)
	ret = append(ret, clientType)
	ret = append(ret, addr...)
	return
}
