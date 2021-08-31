package main

import (
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
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
	"log"
	"os"
)

func main() {
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
	wg := NewWireguard(1500)
	err := stk.CreateNIC(tcpip.NICID(1), (*WireguardEndpoint)(wg))
	if err != nil {
		log.Fatal(err)
	}
	_ = device.NewDevice((*WireguardDevice)(wg), conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, ""))
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
func (w *WireguardDevice) Read(buf []byte, off int) (int, error) {
	view, ok := <-w.buf
	if !ok {
		return 0, os.ErrClosed
	}
	return view.Read(buf[off:])
}

func (w *WireguardDevice) Write(buf []byte, off int) (int, error) {
	pkt := buf[off:]
	if len(pkt) == 0 {
		return 0, nil
	}
	pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{Data: buffer.NewVectorisedView(len(pkt), []buffer.View{buffer.NewViewFromBytes(pkt)})})
	switch pkt[0] >> 4 {
	case 4:
		w.dispatcher.DeliverNetworkPacket("", "", ipv4.ProtocolNumber, pkb)
	case 6:
		w.dispatcher.DeliverNetworkPacket("", "", ipv6.ProtocolNumber, pkb)
	}
	return len(buf), nil
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
	return w.WriteRawPacket(pkt)
}

func (w *WireguardEndpoint) WritePackets(ri stack.RouteInfo, pkts stack.PacketBufferList, pn tcpip.NetworkProtocolNumber) (int, tcpip.Error) {
	curr := pkts.Front()
	count := 0
	for curr != nil {
		err := w.WritePacket(ri, pn, curr)
		if err != nil {
			return count, err
		}
		count++
		curr = curr.Next()
	}
	return count, nil
}

func (w *WireguardEndpoint) WriteRawPacket(pkt *stack.PacketBuffer) tcpip.Error {
	w.buf <- buffer.NewVectorisedView(pkt.Size(), pkt.Views())
	return nil
}
