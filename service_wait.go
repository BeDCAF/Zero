package zero

import (
	"encoding/binary"

	"github.com/sagernet/sing/common/buf"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var _ N.PacketReadWaiter = (*PacketConn)(nil)

func (c *PacketConn) InitializeReadWaiter(options N.ReadWaitOptions) (needCopy bool) {
	c.readWaitOptions = options
	return false
}

func (c *PacketConn) WaitReadPacket() (buffer *buf.Buffer, destination M.Socksaddr, err error) {
	var length uint16
	err = binary.Read(c.Conn, binary.BigEndian, &length)
	if err != nil {
		return nil, M.Socksaddr{}, E.Cause(err, "read chunk length")
	}

	port, err := M.SocksaddrSerializer.ReadPort(c.Conn)
	if err != nil {
		return nil, M.Socksaddr{}, E.Cause(err, "read port")
	}

	destination, err = M.SocksaddrSerializer.ReadAddress(c.Conn)
	if err != nil {
		return nil, M.Socksaddr{}, E.Cause(err, "read addr")
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
