ZERO - a minimal binary proxy protocol.

ZERO was inspired by the Trojan proxy protocol, but was redesigned
to better fit modern technical requirements.

```
[Auth: 16 bytes]      // Opaque 128-bit authentication credential
[CMD: 1 byte]         // Command (CONNECT 0x01; UDP ASSOCIATE 0x03)
[DST.PORT: 2 bytes]   // Desired destination port, in network byte order
[ATYP: 1 byte]        // Address type (IPv4: 0x01; DOMAIN: 0x03; IPv6: 0x04)
[DST.ADDR: N bytes]   // Destination address
[Payload: N bytes]    // Client payload, immediately following the request header
```

Currently, password hash (balake3) is used for authentication.
The fields are similar to SOCKS5, but the order is slightly different.
For a UDP ASSOCIATE request, each UDP datagram is encoded as follows:

```
[LENGTH: 2 bytes] // Payload size
[DST.PORT: 2 bytes]
[ATYP: 1 byte]
[DST.ADDR: N bytes]
[Payload: N bytes]
```


