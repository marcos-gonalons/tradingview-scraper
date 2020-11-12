package tradingview

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
)

// TradingviewSocket ...
type TradingviewSocket struct {
	OnReceiveMarketDataCallback func(symbol string, data *QuoteData)
	OnErrorCallback             func(error)

	conn      *websocket.Conn
	sessionID string
}

// Connect - Connects and returns the trading view socket object
func Connect(
	onReceiveMarketDataCallback func(symbol string, data *QuoteData),
	onErrorCallback func(error),
) (socket SocketInterface, err error) {
	socket = &TradingviewSocket{
		OnReceiveMarketDataCallback: onReceiveMarketDataCallback,
		OnErrorCallback:             onErrorCallback,
	}

	err = socket.Init()

	return
}

// Init connects to the tradingview web socket
func (s *TradingviewSocket) Init() (err error) {
	s.conn, _, err = (&websocket.Dialer{}).Dial("wss://data.tradingview.com/socket.io/websocket", getHeaders())
	if err != nil {
		s.onError(err)
		return
	}

	err = s.checkFirstReceivedMessage()
	if err != nil {
		s.onError(err)
		return
	}

	s.generateSessionID()

	err = s.sendConnectionSetupMessages()
	if err != nil {
		s.onError(err)
		return
	}

	go s.connectionLoop()

	return
}

// Close ...
func (s *TradingviewSocket) Close() (err error) {
	return s.conn.Close()
}

// AddSymbol ...
func (s *TradingviewSocket) AddSymbol(symbol string) (err error) {
	err = s.sendSocketMessage(
		getSocketMessage("quote_add_symbols", []interface{}{s.sessionID, symbol, getFlags()}),
	)
	return
}

// RemoveSymbol ...
func (s *TradingviewSocket) RemoveSymbol(symbol string) (err error) {
	err = s.sendSocketMessage(
		getSocketMessage("quote_remove_symbols", []interface{}{s.sessionID, symbol}),
	)
	return
}

func (s *TradingviewSocket) checkFirstReceivedMessage() (err error) {
	var msg []byte

	_, msg, err = s.conn.ReadMessage()
	if err != nil {
		s.onError(err)
		return
	}

	payload := msg[getPayloadStartingIndex(msg):]
	var p map[string]interface{}

	err = json.Unmarshal(payload, &p)
	if err != nil {
		return
	}

	if p["session_id"] == nil {
		err = errors.New("Cannot recognize the first received message after establishing the connection")
		return
	}

	return
}

func (s *TradingviewSocket) generateSessionID() {
	s.sessionID = "qs_" + GetRandomString(12)
}

func (s *TradingviewSocket) sendConnectionSetupMessages() (err error) {
	messages := []*SocketMessage{
		getSocketMessage("set_auth_token", []string{"unauthorized_user_token"}),
		getSocketMessage("quote_create_session", []string{s.sessionID}),
		getSocketMessage("quote_set_fields", []string{s.sessionID, "lp", "volume", "bid", "ask"}),
	}

	for _, msg := range messages {
		err = s.sendSocketMessage(msg)
		if err != nil {
			return
		}
	}

	return
}

func (s *TradingviewSocket) sendSocketMessage(p *SocketMessage) (err error) {
	payload, _ := json.Marshal(p)
	payloadWithHeader := "~m~" + strconv.Itoa(len(payload)) + "~m~" + string(payload)

	err = s.conn.WriteMessage(websocket.TextMessage, []byte(payloadWithHeader))
	if err != nil {
		s.onError(err)
		return
	}
	return
}

func (s *TradingviewSocket) connectionLoop() {
	var readMsgError error

	for readMsgError == nil {
		var msgType int
		var msg []byte
		msgType, msg, readMsgError = s.conn.ReadMessage()

		if msgType != websocket.TextMessage {
			continue
		}

		if isKeepAliveMsg(msg) {
			err := s.conn.WriteMessage(msgType, msg)
			if err != nil {
				s.onError(err)
				return
			}
			continue
		}

		s.parsePacket(msg)
	}

	s.onError(readMsgError)
}

func (s *TradingviewSocket) parsePacket(packet []byte) {
	index := 0
	for index < len(packet) {
		payloadLength, err := getPayloadLength(packet[index:])
		if err != nil {
			s.onError(err)
			return
		}

		headerLength := 6 + len(strconv.Itoa(payloadLength))
		payload := packet[index+headerLength : index+headerLength+payloadLength]
		index = index + headerLength + len(payload)

		s.parseJSON(payload)
	}
}

func (s *TradingviewSocket) parseJSON(msg []byte) {
	var decodedMessage *SocketMessage
	var err error

	err = json.Unmarshal(msg, &decodedMessage)
	if err != nil {
		s.onError(err)
		return
	}

	if decodedMessage.Message == "critical_error" || decodedMessage.Message == "error" {
		s.onError(errors.New("Error -> " + string(msg)))
		return
	}

	if decodedMessage.Message != "qsd" {
		return
	}

	if decodedMessage.Payload == nil {
		s.onError(errors.New("Msg does not include 'p' -> " + string(msg)))
		return
	}

	p, isPOk := decodedMessage.Payload.([]interface{})
	if !isPOk || len(p) != 2 {
		s.onError(errors.New("There is something wrong with the payload - can't be parsed -> " + string(msg)))
		return
	}

	var decodedQuoteMessage *QuoteMessage
	err = mapstructure.Decode(p[1].(map[string]interface{}), &decodedQuoteMessage)
	if err != nil {
		s.onError(err)
		return
	}

	if decodedQuoteMessage.Status != "ok" || decodedQuoteMessage.Symbol == "" || decodedQuoteMessage.Data == nil {
		s.onError(errors.New("There is something wrong with the payload - couldn't be parsed -> " + string(msg)))
		return
	}

	s.OnReceiveMarketDataCallback(decodedQuoteMessage.Symbol, decodedQuoteMessage.Data)
}

func (s *TradingviewSocket) onError(err error) {
	if s.conn != nil {
		s.conn.Close()
	}
	s.OnErrorCallback(err)
}

func getSocketMessage(m string, p interface{}) *SocketMessage {
	return &SocketMessage{
		Message: m,
		Payload: p,
	}
}

func getFlags() *Flags {
	return &Flags{
		Flags: []string{"force_permission"},
	}
}

func isKeepAliveMsg(msg []byte) bool {
	return string(msg[getPayloadStartingIndex(msg)]) == "~"
}

func getPayloadStartingIndex(msg []byte) int {
	char := ""
	index := 3
	for char != "~" {
		char = string(msg[index])
		index++
	}
	index += 2
	return index
}

func getPayloadLength(msg []byte) (length int, err error) {
	char := ""
	index := 3
	lengthAsString := ""
	for char != "~" {
		char = string(msg[index])
		if char != "~" {
			lengthAsString += char
		}
		index++
	}
	length, err = strconv.Atoi(lengthAsString)
	return
}

func getHeaders() http.Header {
	headers := http.Header{}

	headers.Set("Accept-Encoding", "gzip, deflate, br")
	headers.Set("Accept-Language", "en-US,en;q=0.9,es;q=0.8")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Host", "data.tradingview.com")
	headers.Set("Origin", "https://www.tradingview.com")
	headers.Set("Pragma", "no-cache")
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.193 Safari/537.36")

	return headers
}

