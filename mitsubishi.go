package mc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MitsubishiVersion is an enum type for the different versions of the Mitsubishi protocol.
type MitsubishiVersion int

const (
	Undefined MitsubishiVersion = iota
	A_1E
	Qna_3E
)

type MitsubishiClient struct {
	mu      sync.Mutex
	version MitsubishiVersion
	address string
	port    int
	timeout time.Duration
	conn    net.Conn
}
type MitsubishiMCAddress struct {
	TypeCode     []byte
	BitType      byte
	Format       int
	BeginAddress int
	TypeChar     string
}

func NewMitsubishiClient(version MitsubishiVersion, ip string, port int, timeout time.Duration) (*MitsubishiClient, error) {
	return &MitsubishiClient{
		version: version,
		address: ip,
		port:    port,
		timeout: timeout,
	}, nil
}

func (client *MitsubishiClient) Connect() error {
	if client.conn != nil {
		client.conn.Close()
	}
	address := net.JoinHostPort(client.address, strconv.Itoa(client.port))
	fmt.Println("Connecting to", address)
	conn, err := net.DialTimeout("tcp", address, client.timeout)
	if err != nil {
		return err
	}
	client.conn = conn
	return nil
}

func (client *MitsubishiClient) Connected() bool {
	return client.conn != nil
}

func (client *MitsubishiClient) SendPackageSingle(command []byte) ([]byte, error) {
	if !client.Connected() {
		return nil, errors.New("not connected")
	}
	client.mu.Lock()
	defer client.mu.Unlock()

	fmt.Printf("Sending command :%02x  \n", command)
	_, err := client.conn.Write(command)
	if err != nil {
		return nil, err
	}
	headPackage := make([]byte, 1024)
	n, err := client.conn.Read(headPackage)
	fmt.Println("Reading head package len ", n)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Reading head package %2x \n", headPackage[:n])
	dataPackage := make([]byte, n)

	return append(headPackage[:n], dataPackage...), nil
}

func (client *MitsubishiClient) SendPackageReliable(command []byte) ([]byte, error) {
	if !client.Connected() {
		return nil, errors.New("not connected")
	}
	client.mu.Lock()
	defer client.mu.Unlock()

	_, err := client.conn.Write(command)
	if err != nil {
		return nil, err
	}

	headPackage := make([]byte, 9)
	_, err = client.conn.Read(headPackage)
	if err != nil {
		return nil, err
	}

	contentLength := int(binary.BigEndian.Uint16(headPackage[7:9]))
	dataPackage := make([]byte, contentLength)
	_, err = client.conn.Read(dataPackage)
	if err != nil {
		// Retry once
		client.Connect()
		_, err = client.conn.Write(command)
		if err != nil {
			return nil, err
		}

		_, err = client.conn.Read(headPackage)
		if err != nil {
			return nil, err
		}

		contentLength := int(binary.BigEndian.Uint16(headPackage[7:9]))
		dataPackage := make([]byte, contentLength)
		_, err = client.conn.Read(dataPackage)
		if err != nil {
			return nil, err
		}

		return append(headPackage, dataPackage...), nil
	}
	return append(headPackage, dataPackage...), nil
}

func (client *MitsubishiClient) Close() error {
	if client.conn != nil {
		return client.conn.Close()
	}
	return nil
}

func (client *MitsubishiClient) Read(address string, length uint16, isBit bool) ([]byte, error) {
	var command []byte
	switch client.version {
	case A_1E:
		plcAddress := ConvertArg_A_1E(address)
		command = GetReadCommand_A_1E(plcAddress, length, isBit)
	case Qna_3E:
		plcAddress := ConvertArg_Qna_3E(address)
		command = GetReadCommand_Qna_3E(plcAddress, length, isBit)
	default:
		return nil, errors.New("unknown version")
	}
	responseValue := make([]byte, 0)
	result, err := client.SendPackageSingle(command)
	if err != nil {
		return nil, err
	}
	switch client.version {
	case A_1E:
		responseValue = result[2:]
	case Qna_3E:
		if isBit {
			length = (length + 1) / 2
		}
		responseValue = result[len(result)-int(length):]
	}
	return responseValue, nil
}

func (client *MitsubishiClient) Write(address string, data []byte, isBit bool) ([]byte, error) {
	var command []byte
	switch client.version {
	case A_1E:
		plcAddress := ConvertArg_A_1E(address)
		command = GetWriteCommand_A_1E(plcAddress, data, isBit)
	case Qna_3E:
		command = GetWriteCommand_Qna_3E(address, data, isBit)
	default:
		return nil, errors.New("unknown version")
	}
	return client.SendPackageReliable(command)
}

func GetReadCommand_Qna_3E(address MitsubishiMCAddress, length uint16, isBit bool) []byte {
	addSlice := make([]byte, 8)
	binary.BigEndian.PutUint64(addSlice, uint64(address.BeginAddress))
	ret := make([]byte, 21)
	ret[0] = 0x50
	ret[1] = 0x00
	ret[2] = 0x00
	ret[3] = 0xFF
	ret[4] = 0xFF
	ret[5] = 0x03
	ret[6] = 0x00
	ret[7] = byte((len(ret) - 9) % 256)
	ret[8] = byte((len(ret) - 9) / 256)
	ret[9] = 0x0A
	ret[10] = 0x00
	ret[11] = 0x01
	ret[12] = 0x04
	if isBit {
		ret[13] = 0x01
	} else {
		ret[13] = 0x00
	}
	ret[14] = 0x00
	ret[15] = addSlice[0]
	ret[16] = addSlice[1]
	ret[17] = addSlice[2]
	ret[18] = 0x00
	ret[19] = byte(length % 256)
	ret[20] = byte(length / 256)
	return ret
}

func GetReadCommand_A_1E(address MitsubishiMCAddress, length uint16, isBit bool) []byte {
	if !isBit {
		length = length / 2
	}
	addSlice := make([]byte, 8)
	binary.LittleEndian.PutUint64(addSlice, uint64(address.BeginAddress))
	ret := make([]byte, 12)
	if isBit {
		ret[0] = 0x00
	} else {
		ret[0] = 0x01
	}
	ret[1] = 0xFF
	ret[2] = 0x0A
	ret[3] = 0x00
	ret[4] = addSlice[0]
	ret[5] = addSlice[1]
	ret[6] = 0x00
	ret[7] = 0x00
	ret[8] = address.TypeCode[1]
	ret[9] = address.TypeCode[0]
	ret[10] = byte(length % 256)
	ret[11] = byte(length / 256)
	return ret
}

func GetWriteCommand_Qna_3E(address string, data []byte, isBit bool) []byte {
	ret := make([]byte, 21+len(data))
	ret[0] = 0x50
	ret[1] = 0x00
	ret[2] = 0x00
	ret[3] = 0xFF
	ret[4] = 0xFF
	ret[5] = 0x03
	ret[6] = 0x00
	ret[7] = byte((len(ret) - 9) % 256)
	ret[8] = byte((len(ret) - 9) / 256)
	ret[9] = 0x0A
	ret[10] = 0x00
	ret[11] = 0x01
	ret[12] = 0x14
	if isBit {
		ret[13] = 0x01
	} else {
		ret[13] = 0x00
	}
	ret[14] = 0x00
	ret[15] = byte(address[0])
	ret[16] = byte(address[1])
	ret[17] = byte(address[2])
	ret[18] = 0x00
	ret[19] = byte(len(data) % 256)
	ret[20] = byte(len(data) / 256)
	copy(ret[21:], data)
	return ret
}

func GetWriteCommand_A_1E(address MitsubishiMCAddress, data []byte, isBit bool) []byte {
	length := len(data) / 2
	if isBit {
		length = 1
	}
	addSlice := make([]byte, 8)
	binary.LittleEndian.PutUint64(addSlice, uint64(address.BeginAddress))
	ret := make([]byte, 12+len(data))
	if isBit {
		ret[0] = 0x02
	} else {
		ret[0] = 0x03
	}
	ret[1] = 0xFF
	ret[2] = 0x0A
	ret[3] = 0x00
	ret[4] = addSlice[0]
	ret[5] = addSlice[1]
	ret[6] = 0x00
	ret[7] = 0x00
	ret[8] = address.TypeCode[1]
	ret[9] = address.TypeCode[0]
	ret[10] = byte(length % 256)
	ret[11] = byte(length / 256)
	copy(ret[12:], data)
	return ret
}

func (client *MitsubishiClient) ReadBool(address string) (bool, error) {
	ret, err := client.Read(address, 1, true)
	if err != nil {
		return false, err
	}
	if len(ret) == 0 {
		return false, errors.New("no data")
	}
	value := (ret[0] & 0b00010000) != 0
	return value, nil
}

func (client *MitsubishiClient) ReadInt16(address string) (int16, error) {
	ret, err := client.Read(address, 2, false)
	if err != nil {
		return 0, err
	}
	if len(ret) == 0 {
		return 0, errors.New("no data")
	}
	value := int16(ret[0]) | int16(ret[1])<<8
	return value, nil
}

func (client *MitsubishiClient) ReadUInt16(address string) (uint16, error) {
	ret, err := client.Read(address, 2, false)
	if err != nil {
		return 0, err
	}
	if len(ret) < 2 {
		return 0, errors.New("no data")
	}
	value := uint16(ret[0]) | uint16(ret[1])<<8
	return value, nil
}

func (client *MitsubishiClient) ReadInt32(address string) (int32, error) {
	ret, err := client.Read(address, 4, false)
	if err != nil {
		return 0, err
	}
	if len(ret) < 4 {
		return 0, errors.New("no data")
	}
	value := int32(ret[0]) | int32(ret[1])<<8 | int32(ret[2])<<16 | int32(ret[3])<<24
	return value, nil
}

func (client *MitsubishiClient) ReadUInt32(address string) (uint32, error) {
	ret, err := client.Read(address, 4, false)
	if err != nil {
		return 0, err
	}
	if len(ret) < 4 {
		return 0, errors.New("no data")
	}

	value := uint32(ret[0]) | uint32(ret[1])<<8 | uint32(ret[2])<<16 | uint32(ret[3])<<24
	return value, nil
}

func (client *MitsubishiClient) ReadInt64(address string) (int64, error) {
	ret, err := client.Read(address, 8, false)
	if err != nil {
		return 0, err
	}
	if len(ret) < 8 {
		return 0, errors.New("no data")
	}
	value := int64(ret[0]) | int64(ret[1])<<8 | int64(ret[2])<<16 | int64(ret[3])<<24 |
		int64(ret[4])<<32 | int64(ret[5])<<40 | int64(ret[6])<<48 | int64(ret[7])<<56
	return value, nil
}

func (client *MitsubishiClient) ReadUInt64(address string) (uint64, error) {
	ret, err := client.Read(address, 8, false)
	if err != nil {
		return 0, err
	}
	if len(ret) < 8 {
		return 0, errors.New("no data")
	}
	value := binary.LittleEndian.Uint64(ret)
	return value, nil
}

func ConvertArg_Qna_3E(address string) MitsubishiMCAddress {
	address = strings.ToUpper(address)
	r := []rune(address)
	addressInfo := MitsubishiMCAddress{}

	switch r[0] {
	case 'M':
		addressInfo.TypeCode = []byte{0x90}
		addressInfo.BitType = 0x01
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'X':
		addressInfo.TypeCode = []byte{0x9C}
		addressInfo.BitType = 0x01
		addressInfo.Format = 16
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'Y':
		addressInfo.TypeCode = []byte{0x9D}
		addressInfo.BitType = 0x01
		addressInfo.Format = 16
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'D':
		addressInfo.TypeCode = []byte{0xA8}
		addressInfo.BitType = 0x00
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'W':
		addressInfo.TypeCode = []byte{0xB4}
		addressInfo.BitType = 0x00
		addressInfo.Format = 16
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'L':
		addressInfo.TypeCode = []byte{0x92}
		addressInfo.BitType = 0x01
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'F':
		addressInfo.TypeCode = []byte{0x93}
		addressInfo.BitType = 0x01
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'V':
		addressInfo.TypeCode = []byte{0x94}
		addressInfo.BitType = 0x01
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'B':
		addressInfo.TypeCode = []byte{0xA0}
		addressInfo.BitType = 0x01
		addressInfo.Format = 16
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'R':
		addressInfo.TypeCode = []byte{0xAF}
		addressInfo.BitType = 0x00
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]

	case 'S':
		if r[1] == 'C' {
			addressInfo.TypeCode = []byte{0xC6}
			addressInfo.BitType = 0x01
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		} else if r[1] == 'S' {
			addressInfo.TypeCode = []byte{0xC7}
			addressInfo.BitType = 0x01
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		} else if r[1] == 'N' {
			addressInfo.TypeCode = []byte{0xC8}
			addressInfo.BitType = 0x00
			addressInfo.Format = 100
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		} else {
			addressInfo.TypeCode = []byte{0x98}
			addressInfo.BitType = 0x01
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
			addressInfo.TypeChar = address[:1]
		}

	case 'Z':
		if r[1] == 'R' {
			addressInfo.TypeCode = []byte{0xB0}
			addressInfo.BitType = 0x00
			addressInfo.Format = 16
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		} else {
			addressInfo.TypeCode = []byte{0xCC}
			addressInfo.BitType = 0x00
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
			addressInfo.TypeChar = address[:1]
		}

	case 'T':
		if r[1] == 'N' {
			addressInfo.TypeCode = []byte{0xC2}
			addressInfo.BitType = 0x00
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		} else if r[1] == 'S' {
			addressInfo.TypeCode = []byte{0xC1}
			addressInfo.BitType = 0x01
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		} else if r[1] == 'C' {
			addressInfo.TypeCode = []byte{0xC0}
			addressInfo.BitType = 0x01
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		}

	case 'C':
		if r[1] == 'N' {
			addressInfo.TypeCode = []byte{0xC5}
			addressInfo.BitType = 0x00
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		} else if r[1] == 'S' {
			addressInfo.TypeCode = []byte{0xC4}
			addressInfo.BitType = 0x01
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		} else if r[1] == 'C' {
			addressInfo.TypeCode = []byte{0xC3}
			addressInfo.BitType = 0x01
			addressInfo.Format = 10
			addressInfo.BeginAddress, _ = strconv.Atoi(address[2:])
			addressInfo.TypeChar = address[:2]
		}
	}

	return addressInfo
}

func ConvertArg_A_1E(address string) MitsubishiMCAddress {
	r := []rune(address)
	addressInfo := MitsubishiMCAddress{}
	switch r[0] {
	case 'X':
		addressInfo.TypeCode = []byte{0x58, 0x20}
		addressInfo.BitType = 0x01
		addressInfo.Format = 8
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]
	case 'Y':
		addressInfo.TypeCode = []byte{0x59, 0x20}
		addressInfo.BitType = 0x01
		addressInfo.Format = 16
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]
	case 'M':
		addressInfo.TypeCode = []byte{0x4D, 0x20}
		addressInfo.BitType = 0x01
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]
	case 'S':
		addressInfo.TypeCode = []byte{0x53, 0x20}
		addressInfo.BitType = 0x01
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]
	case 'D':
		addressInfo.TypeCode = []byte{0x44, 0x20}
		addressInfo.BitType = 0x00
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]
	case 'R':
		addressInfo.TypeCode = []byte{0x52, 0x20}
		addressInfo.BitType = 0x00
		addressInfo.Format = 10
		addressInfo.BeginAddress, _ = strconv.Atoi(address[1:])
		addressInfo.TypeChar = address[:1]
	}
	return addressInfo
}
