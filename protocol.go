package zero

import (
	"encoding/binary"
	"net"
	"os"
	"sync"

	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"lukechampine.com/blake3"
)

const (
	KeyLength  = 16
	CommandTCP = 1
	CommandUDP = 3
	CommandMux = 0x7f //Future
)

var _ N.EarlyWriter = (*ClientConn)(nil)

type ClientConn struct {
	N.ExtendedConn
	key           [KeyLength]byte
	destination   M.Socksaddr
	headerWritten bool
}

func NewClientConn(conn net.Conn, key [KeyLength]byte, destination M.Socksaddr) *ClientConn {
	return &ClientConn{
		ExtendedConn: bufio.NewExtendedConn(conn),
		key:          key,
		destination:  destination,
	}
}

func (c *ClientConn) NeedHandshakeForWrite() bool {
	return !c.headerWritten
}

func (c *ClientConn) Write(p []byte) (n int, err error) {
	if c.headerWritten {
		return c.ExtendedConn.Write(p)
	}
	err = ClientHandshake(c.ExtendedConn, c.key, c.destination, p)
	if err != nil {
		return
	}
	n = len(p)
	c.headerWritten = true
	return
}

func (c *ClientConn) WriteBuffer(buffer *buf.Buffer) error {
	if c.headerWritten {
		return c.ExtendedConn.WriteBuffer(buffer)
	}
	err := ClientHandshakeBuffer(c.ExtendedConn, c.key, c.destination, buffer)
	if err != nil {
		return err
	}
	c.headerWritten = true
	return nil
}

func (c *ClientConn) FrontHeadroom() int {
	if !c.headerWritten {
		return KeyLength + 3 + M.MaxSocksaddrLength
	}
	return 0
}

func (c *ClientConn) Upstream() any {
	return c.ExtendedConn
}

func (c *ClientConn) ReaderReplaceable() bool {
	return c.headerWritten
}

func (c *ClientConn) WriterReplaceable() bool {
	return c.headerWritten
}

type ClientPacketConn struct {
	net.Conn
	access          sync.Mutex
	key             [KeyLength]byte
	headerWritten   bool
	readWaitOptions N.ReadWaitOptions
}

func NewClientPacketConn(conn net.Conn, key [KeyLength]byte) *ClientPacketConn {
	return &ClientPacketConn{
		Conn: conn,
		key:  key,
	}
}

func (c *ClientPacketConn) NeedHandshake() bool {
	return !c.headerWritten
}

func (c *ClientPacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	return ReadPacket(c.Conn, buffer)
}

func (c *ClientPacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	if !c.headerWritten {
		c.access.Lock()
		if c.headerWritten {
			c.access.Unlock()
		} else {
			err := ClientHandshakePacket(c.Conn, c.key, destination, buffer)
			if err == nil {
				c.headerWritten = true
			}
			c.access.Unlock()
			return err
		}
	}
	return WritePacket(c.Conn, buffer, destination)
}

func (c *ClientPacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	buffer := buf.With(p)
	destination, err := c.ReadPacket(buffer)
	if err != nil {
		return
	}
	n = buffer.Len()
	if destination.IsDomain() {
		addr = destination
	} else {
		addr = destination.UDPAddr()
	}
	return
}

func (c *ClientPacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return bufio.WritePacket(c, p, addr)
}

func (c *ClientPacketConn) Read(p []byte) (n int, err error) {
	n, _, err = c.ReadFrom(p)
	return
}

func (c *ClientPacketConn) Write(p []byte) (n int, err error) {
	return 0, os.ErrInvalid
}

func (c *ClientPacketConn) FrontHeadroom() int {
	if !c.headerWritten {
		return KeyLength + 7 + 2*M.MaxSocksaddrLength
	}
	return 4 + M.MaxSocksaddrLength
}

func (c *ClientPacketConn) Upstream() any {
	return c.Conn
}

func ClientHandshakeRaw(conn net.Conn, key [KeyLength]byte, command byte, destination M.Socksaddr, payload []byte) error {
	headerLen := KeyLength + 1 + M.SocksaddrSerializer.AddrPortLen(destination)
	header := buf.NewSize(headerLen + len(payload))
	defer header.Release()

	common.Must1(header.Write(key[:]))
	common.Must(header.WriteByte(command))

	err := M.SocksaddrSerializer.WritePort(header, destination.Port)
	if err != nil {
		return err
	}
	err = M.SocksaddrSerializer.WriteAddress(header, destination)
	if err != nil {
		return err
	}
	common.Must1(header.Write(payload))

	_, err = conn.Write(header.Bytes())
	if err != nil {
		return E.Cause(err, "write request")
	}
	return nil
}

func ClientHandshake(conn net.Conn, key [KeyLength]byte, destination M.Socksaddr, payload []byte) error {
	headerLen := KeyLength + M.SocksaddrSerializer.AddrPortLen(destination) + 1
	header := buf.NewSize(headerLen + len(payload))
	defer header.Release()
	common.Must1(header.Write(key[:]))
	common.Must(header.WriteByte(CommandTCP))
	err := M.SocksaddrSerializer.WritePort(header, destination.Port)
	if err != nil {
		return err
	}
	err = M.SocksaddrSerializer.WriteAddress(header, destination)
	if err != nil {
		return err
	}
	common.Must1(header.Write(payload))
	_, err = conn.Write(header.Bytes())
	if err != nil {
		return E.Cause(err, "write request")
	}
	return nil
}

func ClientHandshakeBuffer(conn net.Conn, key [KeyLength]byte, destination M.Socksaddr, payload *buf.Buffer) error {
	header := buf.With(payload.ExtendHeader(KeyLength + M.SocksaddrSerializer.AddrPortLen(destination) + 1))
	common.Must1(header.Write(key[:]))
	common.Must(header.WriteByte(CommandTCP))
	err := M.SocksaddrSerializer.WritePort(header, destination.Port)
	if err != nil {
		return err
	}
	err = M.SocksaddrSerializer.WriteAddress(header, destination)
	if err != nil {
		return err
	}
	_, err = conn.Write(payload.Bytes())
	if err != nil {
		return E.Cause(err, "write request")
	}
	return nil
}

func ClientHandshakePacket(conn net.Conn, key [KeyLength]byte, destination M.Socksaddr, payload *buf.Buffer) error {
	defer payload.Release()
	headerLen := KeyLength + 2*M.SocksaddrSerializer.AddrPortLen(destination) + 3
	payloadLen := payload.Len()
	var header *buf.Buffer
	var writeHeader bool
	if payload.Start() >= headerLen {
		header = buf.With(payload.ExtendHeader(headerLen))
	} else {
		header = buf.NewSize(headerLen)
		defer header.Release()
		writeHeader = true
	}
	common.Must1(header.Write(key[:]))
	common.Must(header.WriteByte(CommandUDP))
	err := M.SocksaddrSerializer.WritePort(header, destination.Port)
	if err != nil {
		return err
	}
	err = M.SocksaddrSerializer.WriteAddress(header, destination)
	if err != nil {
		return err
	}
	common.Must(binary.Write(header, binary.BigEndian, uint16(payloadLen)))
	common.Must(M.SocksaddrSerializer.WritePort(header, destination.Port))
	common.Must(M.SocksaddrSerializer.WriteAddress(header, destination))

	if writeHeader {
		_, err := conn.Write(header.Bytes())
		if err != nil {
			return E.Cause(err, "write request")
		}
	}

	_, err = conn.Write(payload.Bytes())
	if err != nil {
		return E.Cause(err, "write payload")
	}
	return nil
}

func ReadPacket(conn net.Conn, buffer *buf.Buffer) (M.Socksaddr, error) {
	var length uint16
	err := binary.Read(conn, binary.BigEndian, &length)
	if err != nil {
		return M.Socksaddr{}, E.Cause(err, "read chunk length")
	}

	port, err := M.SocksaddrSerializer.ReadPort(conn)
	if err != nil {
		return M.Socksaddr{}, E.Cause(err, "read port")
	}

	destination, err := M.SocksaddrSerializer.ReadAddress(conn)
	if err != nil {
		return M.Socksaddr{}, E.Cause(err, "read addr")
	}
	destination.Port = port

	_, err = buffer.ReadFullFrom(conn, int(length))
	return destination, err
}

func WritePacket(conn net.Conn, buffer *buf.Buffer, destination M.Socksaddr) error {
	defer buffer.Release()
	bufferLen := buffer.Len()
	header := buf.With(buffer.ExtendHeader(M.SocksaddrSerializer.AddrPortLen(destination) + 2))
	common.Must(binary.Write(header, binary.BigEndian, uint16(bufferLen)))
	err := M.SocksaddrSerializer.WritePort(header, destination.Port)
	if err != nil {
		return err
	}
	err = M.SocksaddrSerializer.WriteAddress(header, destination)
	if err != nil {
		return err
	}
	_, err = conn.Write(buffer.Bytes())
	if err != nil {
		return E.Cause(err, "write packet")
	}
	return nil
}

func Key(password string) [KeyLength]byte {
	hasher := blake3.New(16, nil)
	hasher.Write([]byte(password))
	hash := hasher.Sum(nil)
	return *(*[KeyLength]byte)(hash)
}
