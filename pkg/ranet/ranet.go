package main

import (
	"flag"
	"log"

	"gitlab.com/NickCao/RAIT/v4/pkg/rait"
	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"os"
)

var config = flag.String("c", "/etc/rait/rait.conf", "path to config")

func main() {
	flag.Parse()
	r, err := rait.NewRAIT(*config)
	if err != nil {
		log.Fatal(err)
	}
	peers, err := r.Listing()
	if err != nil {
		log.Fatal(err)
	}
	stk := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol, ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			icmp.NewProtocol4, icmp.NewProtocol6,
			udp.NewProtocol, tcp.NewProtocol,
		},
		HandleLocal: true,
	})
	for id := range peers {
		wg := NewWireguard(1500)
		err := stk.CreateNIC(tcpip.NICID(id), (*WireguardEndpoint)(wg))
		if err != nil {
			log.Fatal(err)
		}
	}
}

type Wireguard struct {
	mtu        uint32
	event      chan tun.Event
	buf        chan buffer.VectorisedView
	dispatcher stack.NetworkDispatcher
}

func NewWireguard(mtu uint32) *Wireguard {
	return &Wireguard{
		mtu:   mtu,
		event: make(chan tun.Event),
	}
}

type WireguardEndpoint Wireguard
type WireguardDevice Wireguard

func (w *WireguardDevice) File() *os.File {
	return nil
}
func (w *WireguardDevice) Read([]byte, int) (int, error) {
	return 0, nil
}

func (w *WireguardDevice) Write([]byte, int) (int, error) {
	return 0, nil
}
func (w *WireguardDevice) Flush() error {
	return nil
}
func (w *WireguardDevice) MTU() (int, error) {
	return 0, nil
}
func (w *WireguardDevice) Name() (string, error) {
	return "wireguard", nil
}

func (w *WireguardDevice) Events() chan tun.Event {
	return w.event
}
func (w *WireguardDevice) Close() error {
	close(w.event)
	return nil
}

func (w *WireguardEndpoint) MTU() uint32 {
	return w.mtu
}

func (w *WireguardEndpoint) MaxHeaderLength() uint16 {
	return 0
}

func (w *WireguardEndpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

func (w *WireguardEndpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityNone
}

func (w *WireguardEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	w.dispatcher = dispatcher
}

func (w *WireguardEndpoint) IsAttached() bool {
	return w.dispatcher != nil
}

func (w *WireguardEndpoint) Wait() {
}

func (w *WireguardEndpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

func (w *WireguardEndpoint) AddHeader(local, remote tcpip.LinkAddress, protocol tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) {
}

func (w *WireguardEndpoint) WritePacket(_ stack.RouteInfo, _ tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) tcpip.Error {
	w.buf <- buffer.NewVectorisedView(pkt.Size(), pkt.Views())
	return nil
}

func (w *WireguardEndpoint) WritePackets(ri stack.RouteInfo, pkts stack.PacketBufferList, pn tcpip.NetworkProtocolNumber) (int, tcpip.Error) {
	panic("not implemented")
}

func (w *WireguardEndpoint) WriteRawPacket(pkt *stack.PacketBuffer) tcpip.Error {
	w.buf <- buffer.NewVectorisedView(pkt.Size(), pkt.Views())
	return nil
}
