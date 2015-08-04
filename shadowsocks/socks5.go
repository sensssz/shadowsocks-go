package shadowsocks

import (
	"encoding/binary"
	"errors"
	"net"
)

const ()

const (
	socksVer          = 5
	socksMethodNoAuth = 0
	socksCmdConnect   = 1
	socksReserve      = 0

	typeIPV4       = 1
	typeDomainName = 3
	typeIPV6       = 4

	repSuccess = 0
)

/*
   Handshake request format:
    +----+----------+----------+
    |VER | NMETHODS | METHODS  |
    +----+----------+----------+
    | 1  |    1     | 1 to 255 |
    +----+----------+----------+
    Handshake will be done on the client
    to save a round trip to the server
*/
func handshake(conn net.Conn) error {

	const (
		indexVer = iota
		indexNmethods
		indexMethods
	)

	// 1 byte for version number, 1 byte for number of methods,
	// and at most 256 methods, each taking 1 byte
	max_len := 2 + 256
	// Read request package
	buffer := make([]byte, max_len)
	bytes_read, err = conn.Read(buffer)

	if err != nil {
		return err
	}

	version := buffer[indexVer]
	num_methods := buffer[indexNmethods]

	// The size of the package should be equal to 2 + number of methods
	if bytes_read != 2+num_methods {
		return errors.New("socks NMETHODS does not match size")
	}

	if version != socksVer {
		return errors.New("socks incorrect version")
	}

	// See if the no_auth method is the list of supported methods
	no_auth_supported := false
	for _, method := range buffer[indexMethods:] {
		if method == socksMethodNoAuth {
			no_auth_supported = true
			break
		}
	}

	if !no_auth_supported {
		return errors.New("socks method NO_AUTH not supported by application")
	}

	// Send confirmation response: version 5 and no_auth method
	_, err := conn.Write([]byte{socksVer, socksMethodNoAuth})
	return err
}

/*
   Connect request format:
    +----+-----+-------+------+----------+----------+
    |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
    +----+-----+-------+------+----------+----------+
    | 1  |  1  | X'00' |  1   | Variable |    2     |
    +----+-----+-------+------+----------+----------+
*/
func readAndParseConnectRequest(conn net.Conn) (host string, err error) {
	const (
		indexVer = iota
		indexCmd
		indexRsv
		indexAddrType
		indexAddr
	)
	address_index := indexAddr

	// max length of address is 256 bytes
	max_len := 6 + 256
	buffer := make([]byte, max_len)

	// Read the header to decide the size
	// of the rest of the package
	bytes_read, err := conn.Read(buffer)

	if err != nil {
		return err
	}

	version := buffer[indexVer]
	command := buffer[indexCmd]
	address_type := buffer[indexAddrType]

	// Check the socket version and command
	if version != socksVer {
		err = errors.New("socks incorrect SOCKS version")
		return
	}
	if command != socksCmdConnect {
		err = errors.New("socks command not supported")
		return
	}

	var address_len int
	switch address_type {
	case typeIPV4:
		address_len = net.IPv4len
	case typeIPV6:
		address_len = net.IPv6len
	case typeDomainName:
		address_len = int(buffer[indexAddr])
		address_index++
	default:
		err = errors.New("socks incorrect address type")
		return
	}

	address := buffer[address_index : address_index+address_len]

	switch address_type {
	case typeIPV4 || typeIPV6:
		host = net.IP(address).String()
	case typeDomainName:
		host = string(address)
	}
	port := binary.BigEndian.Uint16(buffer[address_index+address_len : bytes_read])
	host = net.JoinHostPort(host, port)

	return
}
