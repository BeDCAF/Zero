package zero

import (
	"encoding/binary"
	"io"

	"github.com/sagernet/sing/common/buf"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var _ N.PacketReadWaiter = (*ClientPacketConn)(nil)

func (c *ClientPacketConn) InitializeReadWaiter(options N.ReadWaitOptions) (needCopy bool) {
	c.readWaitOptions = options
	return false
}

func (c *ClientPacketConn) WaitReadPacket() (buffer *buf.Buffer, destination M.Socksaddr, err error) {
	var lengthBuf [2]byte
	_, err = io.ReadFull(c.Conn, lengthBuf[:])
	if err != nil {
		return nil, M.Socksaddr{}, E.Cause(err, "read chunk length")
	}
	length := binary.BigEndian.Uint16(lengthBuf[:])

	port, err := M.SocksaddrSerializer.ReadPort(c.Conn)
	if err != nil {
		return nil, M.Socksaddr{}, E.Cause(err, "read destination")
	}
	destination, err = M.SocksaddrSerializer.ReadAddress(c.Conn)
	if err != nil {
		return nil, M.Socksaddr{}, E.Cause(err, "read destination")
	}
	destination.Port = port

	buffer = c.readWaitOptions.NewPacketBuffer()
	_, err = buffer.ReadFullFrom(c.Conn, int(length))
	if err != nil {
		buffer.Release()
		return
	}
	c.readWaitOptions.PostReturn(buffer)
	return
}
