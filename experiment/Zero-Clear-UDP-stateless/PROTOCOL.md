ZERO was inspired by the Trojan proxy protocol (i originally wanted to call it Trojan-Zero), but was redesigned
to better fit modern technical requirements:

```
[Key: 16 bytes]       // Key authentication credentials
[CMD: 1 byte]         // Command (CONNECT 0x01; UDP 0x03)
[ATYP: 1 byte]        // Address type (IPv4: 0x01; DOMAIN: 0x03; IPv6: 0x04)
[DST.ADDR: N bytes]   // Destination address
[DST.PORT: 2 bytes]   // Desired destination port, in network byte order
[Payload: N bytes]    // Client payload, immediately following the request header
```

If the command specifies a UDP and the payload size is greater than zero, then a payload length field is added after the port:

```
[Key: 16 bytes]       // Key authentication credentials
[CMD: 1 byte]         // Command (UDP 0x03)
[ATYP: 1 byte]        // Address type (IPv4: 0x01; DOMAIN: 0x03; IPv6: 0x04)
[DST.ADDR: N bytes]   // Destination address
[DST.PORT: 2 bytes]   // Desired destination port, in network byte order
[LENGTH: 2 bytes]     // Payload size
[Payload: N bytes]    // Client payload, immediately following the request header
```

Currently, password hash (blake3) is used for authentication, but you can accept any 16-byte key.
The ZERO Protocol Specification is available under the [ZERO Protocol License](ZEROProtocolLicense).

This license applies only to the protocol specification and related documentation.
It does not apply to software implementations of the protocol.

Individual implementations may be distributed under separate licenses.
Please refer to the applicable license included with each implementation.