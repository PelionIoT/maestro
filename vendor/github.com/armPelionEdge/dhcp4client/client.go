package dhcp4client

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/armPelionEdge/dhcp4"
)

const (
	MaxDHCPLen = 576
	// progress states
	AtGetOffer                     = 0x01
	AtSendRequest                  = 0x02
	AtGetAcknowledgement           = 0x03
	AtSendRenewalRequest           = 0x04
	AtEndOfRequest                 = 0x05
	AtEndOfRenewal                 = 0x06
	AtGetOfferLoop                 = 0x50
	AtGetAckLoop                   = 0x51
	AtGetOfferLoopTimedOut         = 0xF1
	AtGetOfferError                = 0xF2
	AtGetAckError                  = 0xF3
	AtSendRenewalRequestInitReboot = 0x10
	AtGetOfferUnicast              = 0xA01
	SyscallFailed                  = 0x1001
	// used by GetAcknowledgement()
	Failed   = 0x00
	Success  = int(dhcp4.ACK)
	Rejected = int(dhcp4.NAK)
)

type RequestProgressCB func(state int, addinfo string) (keepgoing bool)

type DhcpRequestOptions struct {
	RequestedParams []byte
	ProgressCB      RequestProgressCB
	// the amount of time we allow for a step in the DHCP communication to
	// take. After this time, we consider the entire operation timedout
	StepTimeout time.Duration
}

func (opts *DhcpRequestOptions) AddRequestParam(code dhcp4.OptionCode) {
	opts.RequestedParams = append(opts.RequestedParams, byte(code))
}

type Client struct {
	hardwareAddr  net.HardwareAddr    //The HardwareAddr to send in the request.
	ignoreServers []net.IP            //List of Servers to Ignore requests from.
	timeout       time.Duration       //Time before we timeout.
	writeTimeout  time.Duration       //Time before we timeout on a write (send)
	broadcast     bool                //Set the Bcast flag in BOOTP Flags
	connection    connection          //The Connection Method to use
	generateXID   func([]byte)        //Function Used to Generate a XID
	opts          *DhcpRequestOptions // optional options
}

//Abstracts the type of underlying socket used
type connection interface {
	Close() error
	Write(packet []byte) error
	ReadFrom() ([]byte, net.IP, error)
	SetReadTimeout(t time.Duration) error
	SetWriteTimeout(t time.Duration) error
}

// Note: The implementor of the original lib chose
// to use a functional programming approach. So 'options'
// are actually functions. Then they made the helper functions
// below return functions. Whew.
// Should be rewritten so as not to be so utterly bizarre at
// some point.

func New(options ...func(*Client) error) (*Client, error) {
	c := Client{
		timeout:      time.Second * 10,
		writeTimeout: time.Second * 10,
		broadcast:    true,
		generateXID:  CryptoGenerateXID,
	}

	err := c.SetOption(options...)
	if err != nil {
		return nil, err
	}

	//if connection hasn't been set as an option create the default.
	if c.connection == nil {
		conn, err := NewInetSock()
		if err != nil {
			return nil, err
		}
		c.connection = conn
	}

	return &c, nil
}

func (c *Client) SetOption(options ...func(*Client) error) error {
	for _, opt := range options {
		if err := opt(c); err != nil {
			return err
		}
	}
	return nil
}

func AuxOpts(opts *DhcpRequestOptions) func(*Client) error {
	return func(c *Client) error {
		if opts != nil {
			c.opts = opts
			if opts.StepTimeout > 0 {
				c.writeTimeout = opts.StepTimeout
				c.timeout = opts.StepTimeout
			}
		}
		return nil
	}
}

func Timeout(t time.Duration) func(*Client) error {
	return func(c *Client) error {
		c.timeout = t
		return nil
	}
}

func WriteTimeout(t time.Duration) func(*Client) error {
	return func(c *Client) error {
		c.writeTimeout = t
		return nil
	}
}

func IgnoreServers(s []net.IP) func(*Client) error {
	return func(c *Client) error {
		c.ignoreServers = s
		return nil
	}
}

func HardwareAddr(h net.HardwareAddr) func(*Client) error {
	return func(c *Client) error {
		c.hardwareAddr = h
		return nil
	}
}

func Broadcast(b bool) func(*Client) error {
	return func(c *Client) error {
		c.broadcast = b
		return nil
	}
}

func Connection(conn connection) func(*Client) error {
	return func(c *Client) error {
		c.connection = conn
		return nil
	}
}

func GenerateXID(g func([]byte)) func(*Client) error {
	return func(c *Client) error {
		c.generateXID = g
		return nil
	}
}

//Close Connections
func (c *Client) Close() error {
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}

//Send the Discovery Packet to the Broadcast Channel
func (c *Client) SendDiscoverPacket(opts *DhcpRequestOptions) (dhcp4.Packet, error) {
	discoveryPacket := c.DiscoverPacket(opts)
	discoveryPacket.PadToMinSize()
	c.connection.SetWriteTimeout(c.writeTimeout)
	return discoveryPacket, c.SendPacket(discoveryPacket)
}

//Send the Discovery Packet to the Broadcast Channel
func (c *Client) SendDiscoverAsUnicastPacket(opts *DhcpRequestOptions) (dhcp4.Packet, error) {
	discoveryPacket := c.DiscoverPacketUnicast(opts)
	discoveryPacket.PadToMinSize()
	c.connection.SetWriteTimeout(c.writeTimeout)
	return discoveryPacket, c.SendPacket(discoveryPacket)
}

//Send the Discovery Packet to the Broadcast Channel
func (c *Client) SendDiscoverPacketExistingIP(opts *DhcpRequestOptions, currentIP net.IP) (dhcp4.Packet, error) {
	discoveryPacket := c.DiscoverPacket(opts)
	discoveryPacket.AddOption(dhcp4.OptionRequestedIPAddress, currentIP.To4())
	discoveryPacket.PadToMinSize()
	c.connection.SetWriteTimeout(c.writeTimeout)
	return discoveryPacket, c.SendPacket(discoveryPacket)
}

//Retreive Offer...
//Wait for the offer for a specific Discovery Packet.
func (c *Client) GetOffer(discoverPacket *dhcp4.Packet) (dhcp4.Packet, error) {
	end := time.Now()
	if c.opts != nil {
		end = end.Add(c.opts.StepTimeout)
	} else {
		end = end.Add(1 * time.Hour)
	}
	for {
		if c.opts != nil && c.opts.ProgressCB != nil {
			c.opts.ProgressCB(AtGetOfferLoop, "")
		}
		now := time.Now()
		if now.After(end) {
			if c.opts != nil && c.opts.ProgressCB != nil {
				c.opts.ProgressCB(AtGetOfferLoopTimedOut, "")
			}
			return nil, errors.New("timeout")
		}
		c.connection.SetReadTimeout(c.timeout)
		readBuffer, source, err := c.connection.ReadFrom()
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				if c.opts != nil && c.opts.ProgressCB != nil {
					c.opts.ProgressCB(AtGetOfferError, fmt.Sprintf("error (timeout): %s", err.Error()))
				}
				return nil, errors.New("timeout")
			}
			if c.opts != nil && c.opts.ProgressCB != nil {
				c.opts.ProgressCB(AtGetOfferError, fmt.Sprintf("error: %s", err.Error()))
			}
			return dhcp4.Packet{}, err
		}

		offerPacket := dhcp4.Packet(readBuffer)
		offerPacketOptions := offerPacket.ParseOptions()

		// Ignore Servers in my Ignore list
		for _, ignoreServer := range c.ignoreServers {
			if source.Equal(ignoreServer) {
				continue
			}

			if offerPacket.SIAddr().Equal(ignoreServer) {
				continue
			}
		}

		if len(offerPacketOptions[dhcp4.OptionDHCPMessageType]) < 1 || dhcp4.MessageType(offerPacketOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.Offer || !bytes.Equal(discoverPacket.XId(), offerPacket.XId()) {
			continue
		}

		return offerPacket, nil
	}

}

//Send Request Based On the offer Received.
//func (c *Client) SendRequest(offerPacket *dhcp4.Packet, opts *DhcpRequestOptions) (dhcp4.Packet, error) {
func (c *Client) SendRequest(offerPacket *dhcp4.Packet) (dhcp4.Packet, error) {
	requestPacket := c.RequestPacket(offerPacket)
	requestPacket.PadToMinSize()
	c.connection.SetWriteTimeout(c.writeTimeout)
	return requestPacket, c.SendPacket(requestPacket)
}

//Retreive Acknowledgement
//Wait for the offer for a specific Request Packet.
func (c *Client) GetAcknowledgement(requestPacket *dhcp4.Packet) (pack dhcp4.Packet, ackval int, err error) {
	end := time.Now()
	if c.opts != nil {
		end = end.Add(c.opts.StepTimeout)
	} else {
		end = end.Add(1 * time.Hour)
	}
	for {
		if c.opts != nil && c.opts.ProgressCB != nil {
			c.opts.ProgressCB(AtGetAckLoop, "")
		}
		now := time.Now()
		if now.After(end) {
			if c.opts != nil && c.opts.ProgressCB != nil {
				c.opts.ProgressCB(AtGetOfferLoopTimedOut, "")
			}
			err = errors.New("timeout")
			return
		}
		err = c.connection.SetReadTimeout(c.timeout)
		if err != nil {
			if c.opts != nil && c.opts.ProgressCB != nil {
				c.opts.ProgressCB(SyscallFailed, fmt.Sprintf("timeout failed: %s", err.Error()))
			}
			// ignore this error
			err = nil
		}
		readBuffer, source, err2 := c.connection.ReadFrom()
		if err2 != nil {
			if err2, ok := err2.(net.Error); ok && err2.Timeout() {
				if c.opts != nil && c.opts.ProgressCB != nil {
					c.opts.ProgressCB(AtGetAckError, fmt.Sprintf("error (timeout): %s", err2.Error()))
				}
				err = errors.New("timeout")
				return
			}
			if c.opts != nil && c.opts.ProgressCB != nil {
				c.opts.ProgressCB(AtGetAckError, fmt.Sprintf("error: %s", err.Error()))
			}
			pack = dhcp4.Packet{}
			err = err2
			return
		}

		acknowledgementPacket := dhcp4.Packet(readBuffer)
		acknowledgementPacketOptions := acknowledgementPacket.ParseOptions()

		// Ignore Servers in my Ignore list
		for _, ignoreServer := range c.ignoreServers {
			if source.Equal(ignoreServer) {
				continue
			}

			if acknowledgementPacket.SIAddr().Equal(ignoreServer) {
				continue
			}
		}

		if !bytes.Equal(requestPacket.XId(), acknowledgementPacket.XId()) || len(acknowledgementPacketOptions[dhcp4.OptionDHCPMessageType]) < 1 || (dhcp4.MessageType(acknowledgementPacketOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.ACK && dhcp4.MessageType(acknowledgementPacketOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.NAK) {
			continue
		}

		if dhcp4.MessageType(acknowledgementPacketOptions[dhcp4.OptionDHCPMessageType][0]) == dhcp4.NAK {
			ackval = Rejected
		}
		if dhcp4.MessageType(acknowledgementPacketOptions[dhcp4.OptionDHCPMessageType][0]) == dhcp4.ACK {
			ackval = Success
		}

		pack = acknowledgementPacket
		err = nil
		return
	}
}

//Send Decline to the received acknowledgement.
func (c *Client) SendDecline(acknowledgementPacket *dhcp4.Packet) (dhcp4.Packet, error) {
	declinePacket := c.DeclinePacket(acknowledgementPacket)
	declinePacket.PadToMinSize()
	c.connection.SetWriteTimeout(c.writeTimeout)
	return declinePacket, c.SendPacket(declinePacket)
}

//Send a DHCP Packet.
func (c *Client) SendPacket(packet dhcp4.Packet) error {
	// SetWriteDeadline ?
	return c.connection.Write(packet)
}

//Create Discover Packet
func (c *Client) DiscoverPacket(opts *DhcpRequestOptions) dhcp4.Packet {
	messageid := make([]byte, 4)
	c.generateXID(messageid)

	packet := dhcp4.NewPacket(dhcp4.BootRequest)
	packet.SetCHAddr(c.hardwareAddr)
	packet.SetXId(messageid)
	packet.SetBroadcast(c.broadcast)

	packet.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Discover)})

	if opts != nil {
		if len(opts.RequestedParams) > 0 {
			packet.AddOption(dhcp4.OptionParameterRequestList, opts.RequestedParams)
		}
	}
	//packet.PadToMinSize()
	return packet
}

//Create Discover Packet
func (c *Client) DiscoverPacketUnicast(opts *DhcpRequestOptions) dhcp4.Packet {
	messageid := make([]byte, 4)
	c.generateXID(messageid)

	packet := dhcp4.NewPacket(dhcp4.BootRequest)
	packet.SetCHAddr(c.hardwareAddr)
	packet.SetXId(messageid)
	packet.SetSIAddr(net.IPv4(0, 0, 0, 0))
	//	packet.SetBroadcast(c.broadcast)

	packet.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Discover)})
	packet.AddOption(dhcp4.OptionClientIdentifier, dhcp4.MakeClientIdentifier(dhcp4.ClientIdentifierEthernet, c.hardwareAddr))

	if opts != nil {
		if len(opts.RequestedParams) > 0 {
			packet.AddOption(dhcp4.OptionParameterRequestList, opts.RequestedParams)
		}
	}
	//packet.PadToMinSize()
	return packet
}

//Create Request Packet
func (c *Client) RequestPacket(offerPacket *dhcp4.Packet) dhcp4.Packet {
	offerOptions := offerPacket.ParseOptions()

	packet := dhcp4.NewPacket(dhcp4.BootRequest)
	packet.SetCHAddr(c.hardwareAddr)

	packet.SetXId(offerPacket.XId())
	packet.SetCIAddr(offerPacket.CIAddr())
	packet.SetSIAddr(net.IPv4(0, 0, 0, 0))
	//packet.SetSIAddr(offerPacket.SIAddr())

	//	packet.SetBroadcast(c.broadcast)
	packet.AddOption(dhcp4.OptionClientIdentifier, dhcp4.MakeClientIdentifier(dhcp4.ClientIdentifierEthernet, offerPacket.CHAddr()))
	packet.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Request)})
	packet.AddOption(dhcp4.OptionRequestedIPAddress, (offerPacket.YIAddr()).To4())
	packet.AddOption(dhcp4.OptionServerIdentifier, offerOptions[dhcp4.OptionServerIdentifier])

	if c.opts != nil {
		if len(c.opts.RequestedParams) > 0 {
			packet.AddOption(dhcp4.OptionParameterRequestList, c.opts.RequestedParams)
		}
	}

	return packet
}

//Create Request Packet For a Renew
func (c *Client) RenewalRequestPacket(acknowledgement *dhcp4.Packet, opts *DhcpRequestOptions) dhcp4.Packet {
	messageid := make([]byte, 4)
	c.generateXID(messageid)

	acknowledgementOptions := acknowledgement.ParseOptions()

	packet := dhcp4.NewPacket(dhcp4.BootRequest)
	packet.SetCHAddr(acknowledgement.CHAddr())

	packet.SetXId(messageid)
	packet.SetCIAddr(acknowledgement.YIAddr())
	//	packet.SetSIAddr(acknowledgement.SIAddr())
	packet.SetSIAddr(net.IPv4(0, 0, 0, 0))

	//	packet.SetBroadcast(c.broadcast)
	packet.AddOption(dhcp4.OptionClientIdentifier, dhcp4.MakeClientIdentifier(dhcp4.ClientIdentifierEthernet, acknowledgement.CHAddr()))
	packet.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Request)})
	packet.AddOption(dhcp4.OptionRequestedIPAddress, (acknowledgement.YIAddr()).To4())
	packet.AddOption(dhcp4.OptionServerIdentifier, acknowledgementOptions[dhcp4.OptionServerIdentifier])

	if opts != nil {
		if len(opts.RequestedParams) > 0 {
			packet.AddOption(dhcp4.OptionParameterRequestList, opts.RequestedParams)
		}
	}

	return packet
}

// RenewalRequestPacketInitReboot - creates a Request packet for the INIT-REBOOT state
// as defined in RFC2131
func (c *Client) RenewalRequestPacketInitReboot(currentIP net.IP, opts *DhcpRequestOptions) dhcp4.Packet { // acknowledgement *dhcp4.Packet,
	messageid := make([]byte, 4)
	c.generateXID(messageid)

	//	acknowledgementOptions := acknowledgement.ParseOptions()

	packet := dhcp4.NewPacket(dhcp4.BootRequest)
	packet.SetCHAddr(c.hardwareAddr)

	packet.SetXId(messageid)
	// packet.SetCIAddr(acknowledgement.YIAddr())
	packet.SetSIAddr(net.IPv4(0, 0, 0, 0))

	//	packet.SetBroadcast(c.broadcast)
	packet.AddOption(dhcp4.OptionClientIdentifier, dhcp4.MakeClientIdentifier(dhcp4.ClientIdentifierEthernet, c.hardwareAddr))
	packet.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Request)})
	packet.AddOption(dhcp4.OptionRequestedIPAddress, currentIP.To4())

	if opts != nil {
		if len(opts.RequestedParams) > 0 {
			packet.AddOption(dhcp4.OptionParameterRequestList, opts.RequestedParams)
		}
	}

	return packet
}

//Create Release Packet For a Release
func (c *Client) ReleasePacket(acknowledgement *dhcp4.Packet) dhcp4.Packet {
	messageid := make([]byte, 4)
	c.generateXID(messageid)

	acknowledgementOptions := acknowledgement.ParseOptions()

	packet := dhcp4.NewPacket(dhcp4.BootRequest)
	packet.SetCHAddr(acknowledgement.CHAddr())

	packet.SetXId(messageid)
	packet.SetCIAddr(acknowledgement.YIAddr())

	packet.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Release)})
	packet.AddOption(dhcp4.OptionServerIdentifier, acknowledgementOptions[dhcp4.OptionServerIdentifier])

	return packet
}

//Create Decline Packet
func (c *Client) DeclinePacket(acknowledgement *dhcp4.Packet) dhcp4.Packet {
	messageid := make([]byte, 4)
	c.generateXID(messageid)

	acknowledgementOptions := acknowledgement.ParseOptions()

	packet := dhcp4.NewPacket(dhcp4.BootRequest)
	packet.SetCHAddr(acknowledgement.CHAddr())
	packet.SetXId(messageid)

	packet.AddOption(dhcp4.OptionDHCPMessageType, []byte{byte(dhcp4.Decline)})
	packet.AddOption(dhcp4.OptionRequestedIPAddress, (acknowledgement.YIAddr()).To4())
	packet.AddOption(dhcp4.OptionServerIdentifier, acknowledgementOptions[dhcp4.OptionServerIdentifier])

	return packet
}

// InitRebootRequest is called when we want to use an existing IP, and just try to send a DISCOVER pack
// with out exising IP. What happens next depends on if the DHCP server is "authoritive" or not.
// If it is, and we have asked for an IP it will nto provide, then we will get a deny answer,
// otherwise we should wait for the timeout, and return this back. In this case a fresh
// Request should be sent out.
func (c *Client) InitRebootRequest(currentIP net.IP, opts *DhcpRequestOptions) (int, dhcp4.Packet, error) {
	c.opts = opts
	renewRequest := c.RenewalRequestPacketInitReboot(currentIP, opts)
	renewRequest.PadToMinSize()
	c.connection.SetWriteTimeout(c.writeTimeout)
	keepgoing := true
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtSendRenewalRequestInitReboot, "")
	}
	if !keepgoing {
		return Failed, nil, nil
	}
	err := c.SendPacket(renewRequest)
	if err != nil {
		return Failed, renewRequest, err
	}
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtGetAcknowledgement, "")
	}
	if !keepgoing {
		return Failed, nil, nil
	}
	newAcknowledgement, ackval, err := c.GetAcknowledgement(&renewRequest)
	if err != nil {
		return Failed, newAcknowledgement, err
	}
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtEndOfRenewal, "")
	}

	return ackval, newAcknowledgement, nil
}

// Request will do a Full DHCP Request. If a broadcast discover fails, we will try a
// unicast discover
func (c *Client) Request(opts *DhcpRequestOptions) (int, dhcp4.Packet, error) {
	c.opts = opts
	discoveryPacket, err := c.SendDiscoverPacket(opts)
	keepgoing := true
	if err != nil {
		return Failed, discoveryPacket, err
	}
	//	fmt.Printf("@GetOffer\n")
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtGetOffer, "")
	}
	if !keepgoing {
		return Failed, discoveryPacket, nil
	}
	offerPacket, err := c.GetOffer(&discoveryPacket)
	if err != nil {
		if err.Error() == "timeout" {
			// ok - let's try a unicast discover
			keepgoing = opts.ProgressCB(AtGetOfferUnicast, "")
			discoveryPacket, err = c.SendDiscoverAsUnicastPacket(opts)
			if err != nil || !keepgoing {
				return Failed, discoveryPacket, err
			}
			offerPacket, err = c.GetOffer(&discoveryPacket)
			if err != nil {
				return Failed, offerPacket, err
			}
		} else {
			return Failed, offerPacket, err
		}
	}
	//	fmt.Printf("@SendRequest\n")
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtSendRequest, "")
	}
	if !keepgoing {
		return Failed, offerPacket, nil
	}
	requestPacket, err := c.SendRequest(&offerPacket)
	if err != nil {
		return Failed, requestPacket, err
	}
	//	fmt.Printf("@GetAcknowledgement\n")
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtGetAcknowledgement, "")
	}
	if !keepgoing {
		return Failed, requestPacket, nil
	}
	acknowledgement, ackval, err := c.GetAcknowledgement(&requestPacket)
	if err != nil {
		return Failed, acknowledgement, err
	}
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtEndOfRequest, "")
	}
	//acknowledgementOptions := acknowledgement.ParseOptions()
	// if dhcp4.MessageType(acknowledgementOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.ACK {
	// 	return ackval, acknowledgement, nil
	// }
	return ackval, acknowledgement, nil
}

// RequestAsUnicast does a normal request process, but starts with a Unicast flag DISCOVER packet
// this is actually what udhcp does on Linux. Apparently, some routers have poor DHCP server implementations
// or don't like broadcast flag and/or a broadcast Ethernet frame for some reason.
func (c *Client) RequestAsUnicast(opts *DhcpRequestOptions) (int, dhcp4.Packet, error) {
	c.opts = opts
	discoveryPacket, err := c.SendDiscoverAsUnicastPacket(opts)
	keepgoing := true
	if err != nil {
		return Failed, discoveryPacket, err
	}
	//	fmt.Printf("@GetOffer\n")
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtGetOffer, "")
	}
	if !keepgoing {
		return Failed, discoveryPacket, nil
	}
	offerPacket, err := c.GetOffer(&discoveryPacket)
	if err != nil {
		return Failed, offerPacket, err
	}
	//	fmt.Printf("@SendRequest\n")
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtSendRequest, "")
	}
	if !keepgoing {
		return Failed, offerPacket, nil
	}
	requestPacket, err := c.SendRequest(&offerPacket)
	if err != nil {
		return Failed, requestPacket, err
	}
	//	fmt.Printf("@GetAcknowledgement\n")
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtGetAcknowledgement, "")
	}
	if !keepgoing {
		return Failed, requestPacket, nil
	}
	acknowledgement, ackval, err := c.GetAcknowledgement(&requestPacket)
	if err != nil {
		return Failed, acknowledgement, err
	}
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtEndOfRequest, "")
	}
	//acknowledgementOptions := acknowledgement.ParseOptions()
	// if dhcp4.MessageType(acknowledgementOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.ACK {
	// 	return ackval, acknowledgement, nil
	// }
	return ackval, acknowledgement, nil
}

//Renew a lease backed on the Acknowledgement Packet.
//Returns Sucessfull, The AcknoledgementPacket, Any Errors
func (c *Client) Renew(acknowledgement dhcp4.Packet, opts *DhcpRequestOptions) (int, dhcp4.Packet, error) {
	c.opts = opts
	renewRequest := c.RenewalRequestPacket(&acknowledgement, nil)
	renewRequest.PadToMinSize()
	c.connection.SetWriteTimeout(c.writeTimeout)
	keepgoing := true
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtSendRenewalRequest, "")
	}
	if !keepgoing {
		return Failed, nil, nil
	}
	err := c.SendPacket(renewRequest)
	if err != nil {
		return Failed, renewRequest, err
	}
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtGetAcknowledgement, "")
	}
	if !keepgoing {
		return Failed, nil, nil
	}
	newAcknowledgement, ackval, err := c.GetAcknowledgement(&renewRequest)
	if err != nil {
		return ackval, newAcknowledgement, err
	}
	if opts != nil && opts.ProgressCB != nil {
		keepgoing = opts.ProgressCB(AtEndOfRenewal, "")
	}
	//	newAcknowledgementOptions := newAcknowledgement.ParseOptions()
	// if dhcp4.MessageType(newAcknowledgementOptions[dhcp4.OptionDHCPMessageType][0]) != dhcp4.ACK {
	// 	return false, newAcknowledgement, nil
	// }

	return ackval, newAcknowledgement, nil
}

//Release a lease backed on the Acknowledgement Packet.
//Returns Any Errors
func (c *Client) Release(acknowledgement dhcp4.Packet) error {
	release := c.ReleasePacket(&acknowledgement)
	release.PadToMinSize()
	c.connection.SetWriteTimeout(c.writeTimeout)
	return c.SendPacket(release)
}
