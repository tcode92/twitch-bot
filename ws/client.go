package ws

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
)

const ContinuationOpcode = 0x0
const TextOpcode = 0x1
const BinaryOpcode = 0x2
const CloseOpcode = 0x8
const PingOpcode = 0x9
const PongOpcode = 0xA

type Client struct {
	url             *url.URL
	OnTextMessage   func(message string)
	OnBinaryMessage func(message []byte)
	OnPing          func()
	OnPong          func()
	OnDisconnect    func()
	Conn            net.Conn
	closeChan       chan error
}

// returns a new ws client
func NewClient(wsUrl string) (*Client, error) {
	// validate url
	u, err := url.Parse(wsUrl)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, errors.New(`invalid url schema: expected "ws" or "wss"`)
	}

	// return client
	c := &Client{
		url:       u,
		closeChan: make(chan error),
	}

	return c, nil
}

// initialize the connection with the server
//
// returns error on failure or nil on success
func (c *Client) Connect() error {
	// handshake request
	webSecKey := genSecWebSocketKey()
	hsReq := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"\r\n", c.url.Path, c.url.Host, webSecKey)
	// connect to the server
	var conn net.Conn
	var err error
	if c.url.Scheme == "wss" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
		}
		conn, err = tls.Dial("tcp", c.url.Host, tlsConfig)
	} else {
		conn, err = net.Dial("tcp", c.url.Host)

	}
	if err != nil {
		conn.Close()
		return fmt.Errorf("error in dial: %w", err)
	}

	// send handshake request
	_, err = conn.Write([]byte(hsReq))
	if err != nil {
		conn.Close()
		return fmt.Errorf("error in http req: %w", err)
	}

	// read handshake response
	response := make([]byte, 1024)
	_, err = conn.Read(response)
	if err != nil {
		conn.Close()
		// maybe return our own errors?
		return err
	}
	acceptKey, err := getAcceptKeyFromHeaders(string(response))
	if err != nil {
		conn.Close()
		return err
	}

	// validate accept key
	if acceptKey != computeAcceptKey(webSecKey) {
		conn.Close()
		return errors.New("recived invalid accept key")
	}

	// set the connection on client
	c.Conn = conn

	// handle incoming messages
	go c.handleIncomingMessages()

	return nil
}

// closes the tcp connection
func (c *Client) Close() {
	if c.Conn != nil {
		// send signal to chan
		c.closeChan <- nil
		// close the channel
		close(c.closeChan)

		// the connection should be closed bye the goroutine

		// set conection to nil
		c.Conn = nil

	}
}

func (c *Client) SendText(t string) error {
	// text is bytes anyway, we convert it to byets and call send
	return c.send([]byte(t), TextOpcode)
}

func (c *Client) SendBytes(b []byte) error {
	return c.send(b, BinaryOpcode)

}

func (c *Client) SendJson(data interface{ any }) error {
	// marshal json and send the message
	var j []byte
	j, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.send(j, TextOpcode)
}

func (c *Client) SendJsonBin(data interface{ any }) error {
	// marshal json and send the message
	var j []byte
	j, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.send(j, BinaryOpcode)
}

// opcode can be "t" for text and "b" for binary
// should we ignore text and send all data as binary anyway?
// i think it's required and correct to send the correct opcode to server/client so they can handle them accordingly
func (c *Client) send(b []byte, opcode byte) error {
	payloadLen := len(b)
	// should we send chunked messages?

	var frameSize = 0
	payloadStart := 0
	if payloadLen < 126 {
		// payload size is <= 125 so we have
		// 1byte for fin+reserved
		// 1byte for mask(bit) and payload length
		// 4byte for mask (32bit)
		// and the payload data
		frameSize = 6 + payloadLen
	} else if payloadLen == 126 {
		// same as above but we have 2 extra bytes to write the payload length
		frameSize = 8 + payloadLen
	} else {
		// same as above but we have 8 extra bytes to write the payload length
		frameSize = 14 + payloadLen
	}
	payload := make([]byte, frameSize)
	// write first byte fin+opcode
	if opcode == TextOpcode {
		payload[0] = 0b10000001
	} else {
		payload[0] = 0b10000010
	}
	// set mask and payload length
	mask := getMaskKey()

	if payloadLen < 126 {
		// len
		payload[1] = 0b10000000 | byte(payloadLen)
		// mask
		copy(payload[2:6], mask)
		payloadStart = 6
	} else if payloadLen == 126 {
		// len
		payload[1] = 0b11111110
		binary.BigEndian.PutUint16(payload[2:4], uint16(payloadLen))
		// mask
		copy(payload[4:8], mask)
		payloadStart = 8
	} else {
		// len
		payload[1] = 0b11111111
		binary.BigEndian.PutUint64(payload[2:10], uint64(payloadLen))
		// mask
		copy(payload[10:14], mask)
		payloadStart = 14
	}
	// mask the payload and write it
	for i, data := range b {
		payload[payloadStart+i] = data ^ mask[i%4]
	}
	// write the frame to the connection
	_, err := c.Conn.Write(payload)
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) SendPing() error {
	m := getMaskKey()
	pingMsg := []byte{
		0b10001001,             // Fin + Opcode (Ping) 0x9
		0b10000100,             // mask must be set from the client + Payload len
		m[0], m[1], m[2], m[3], // mask
		0b1010000 ^ m[0], 0b1101001 ^ m[1], 0b1101110 ^ m[2], 0b1100111 ^ m[3], // "Ping" payload masked
	}
	_, err := c.Conn.Write(pingMsg)
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) SendPong() error {
	mask := getMaskKey()
	pongMsg := []byte{
		0b10001010,                         // Fin + Opcode (Pong) 0x10
		0b10000100,                         // Mask must be set from the client + Payload len
		mask[0], mask[1], mask[2], mask[3], // Mask
		0b1010000, 0b1101111, 0b1101110, 0b1100111, // "Pong" payload
	}
	_, err := c.Conn.Write(pongMsg)
	if err != nil {
		return err
	}
	return nil
}

// generates 4 byte random mask key to mask
// the payload sent to the server
func getMaskKey() []byte {
	key := make([]byte, 4)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return key
}

// generates a random websocket key for the handshake
func genSecWebSocketKey() string {
	key := make([]byte, 16)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func computeAcceptKey(key string) string {
	const websocketMagic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	_, err := h.Write([]byte(string(key) + websocketMagic))
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// parse handshake response headers
//
// return Sec-Accept-Key header if present or error
func getAcceptKeyFromHeaders(headers string) (string, error) {
	lines := strings.Split(headers, "\r\n")
	if len(lines) == 0 {
		return "", errors.New("invalid headers")
	}
	// check the response status is 101
	// http header is (http version | http response status | http status text)
	result := strings.Split(lines[0], " ")
	status := result[1]
	if status != "101" {
		return "", fmt.Errorf("invalid status: expected 101 recived %s", status)
	}
	var secAcceptKey string
	for i, header := range lines {
		// we parsed the first line already
		if i == 0 {
			continue
		}
		if strings.HasPrefix(header, "Sec-WebSocket-Accept: ") {
			secAcceptKey = strings.TrimSpace(strings.TrimPrefix(header, "Sec-WebSocket-Accept: "))
			break
		}
	}
	if secAcceptKey == "" {
		return secAcceptKey, errors.New("sec-websocket-accept not found")
	}
	return secAcceptKey, nil
}

// client default ping and pong functions
// user can implement them if he wants to
// they are not required.

// needs implementation

// lets se if i can calculate hexadecimal by hand

// what is the value in bits and int for 0x7D

// 15 14 13 12 11 10 9  8  7  6  5   4   3   2   1  0
// F  E  D  C  B  A  9  8  7  6  5   4   3   2   1  0
//       1                 1
//	13 x 16^1    +    7 x 16^0     =  125 (seems easy)

func (c *Client) handleIncomingMessages() {
	defer func() {
		if c.Conn != nil {
			c.OnDisconnect()
			c.Conn.Close()
		}
		close(c.closeChan)
	}()
	var message bytes.Buffer
	var opcode uint8
	conn := c.Conn
messageHandler:
	for {
		select {
		case err := <-c.closeChan:
			{
				if err != nil {
					// error occured
					println(err)
				}
				break messageHandler
			}
		default:
			// read first 2 bytes to determin message type and length
			header := make([]byte, 2)
			_, err := conn.Read(header)
			if err != nil {
				// if err is EOF that means that the server closed the connection. we should return.
				if err == io.EOF {
					c.closeChan <- nil
					return
				}
				// error reading from the connection, should close.
				c.closeChan <- err
				return
			}
			fin := (header[0] & 0b10000000) != 0
			frameOpcode := header[0] & 0b00001111 // current frame's opcode

			// If this is the first frame of the message, set the opcode
			if opcode == 0 && frameOpcode != ContinuationOpcode {
				opcode = frameOpcode
			}

			// If it's not a continuation frame, but we're already processing a fragmented message, it's an error
			if frameOpcode != ContinuationOpcode && opcode != frameOpcode {
				println("Received fragmented message with mismatched opcodes")
				return
			}

			// close connection if mask is set, server should always send unmasked frames.
			if (header[1] & 0b10000000) != 0 {
				// break will exit the loop and defer conn.Close() is called.
				return
			}

			payloadLen := int(header[1] & 0b01111111)

			// determing if payload is extended or it's full.

			if payloadLen == 126 {
				// payload length is extended to the next 2 byets
				extended := make([]byte, 2)
				_, err := conn.Read(extended)
				if err != nil {
					// error reading from the connection, should close.
					println("Error reading from the connection", err)
					return
				}
				payloadLen = int(binary.BigEndian.Uint16(extended))
			} else if payloadLen == 127 {
				// payload length is extended to the next 8 byets
				extended := make([]byte, 8)
				_, err := conn.Read(extended)
				if err != nil {
					// error reading from the connection, should close.
					println("Error reading from the connection", err)
					return
				}
				payloadLen = int(binary.BigEndian.Uint64(extended))
			}

			// read payload
			p := make([]byte, payloadLen)
			_, err = conn.Read(p)
			if err != nil {
				// error reading from the connection, should close.
				println("Error reading from the connection", err)
				return
			}
			_, err = message.Write(p)
			if err != nil {
				// error reading from the connection, should close.
				println("Error reading from the connection", err)
				return
			}

			if fin {
				buffer := message.Bytes()
				// based on opcode call the specific user callback

				switch opcode {
				case TextOpcode:
					if c.OnTextMessage != nil {
						c.OnTextMessage(string(buffer))
					}
				case BinaryOpcode:
					if c.OnBinaryMessage != nil {
						c.OnBinaryMessage(buffer)
					}
				case PingOpcode:
					c.SendPong()
				case PongOpcode:
					// Handle pong response, if needed
					fmt.Println("Recived pong")
				default:
					fmt.Println("Unknown opcode:", opcode)
				}

				// reset buffer and opcode
				message.Reset()
				opcode = 0
			}

		}
	}

}
