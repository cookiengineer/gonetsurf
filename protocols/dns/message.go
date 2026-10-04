// Package dns implements DNS message encoding, a caching resolver, and
// DNS-over-TLS and DNS-over-HTTPS transports.
//
// See specifications/dns/rfc1034.txt, rfc1035.txt, rfc2181.txt, rfc6891.txt,
// rfc7858.txt, and rfc8484.txt.
package dns

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// ErrMalformed is returned for malformed DNS messages.
var ErrMalformed = errors.New("dns: malformed message")

// Type is a DNS record type.
type Type uint16

// Record types.
const (
	TypeA     Type = 1
	TypeNS    Type = 2
	TypeCNAME Type = 5
	TypeSOA   Type = 6
	TypeMX    Type = 15
	TypeTXT   Type = 16
	TypeAAAA  Type = 28
	TypeSRV   Type = 33
	TypeOPT   Type = 41
)

// Class is a DNS record class.
type Class uint16

// Record classes.
const (
	ClassIN Class = 1
)

// Header flag bits.
const (
	FlagResponse           = 1 << 15
	FlagAuthoritative      = 1 << 10
	FlagTruncated          = 1 << 9
	FlagRecursionDesired   = 1 << 8
	FlagRecursionAvailable = 1 << 7
)

// Response codes.
const (
	RCodeSuccess        = 0
	RCodeFormatError    = 1
	RCodeServerFailure  = 2
	RCodeNameError      = 3
	RCodeNotImplemented = 4
	RCodeRefused        = 5
)

// Question is a DNS question.
type Question struct {
	Name  string
	Type  Type
	Class Class
}

// RR is a DNS resource record.
type RR struct {
	Name     string
	Type     Type
	Class    Class
	TTL      uint32
	Data     []byte
	Target   string
	Address  netip.Addr
	Text     []string
	Priority uint16
	Weight   uint16
	Port     uint16
}

// Message is a DNS message.
type Message struct {
	ID          uint16
	Flags       uint16
	Questions   []Question
	Answers     []RR
	Authorities []RR
	Additionals []RR
}

// ResponseCode returns the message's response code.
func (message *Message) ResponseCode() int {
	return int(message.Flags & 0x0F)
}

// IsResponse reports whether the QR bit is set.
func (message *Message) IsResponse() bool {
	return message.Flags&FlagResponse != 0
}

// ParseMessage decodes a DNS message.
func ParseMessage(data []byte) (*Message, error) {
	if len(data) < 12 {
		return nil, ErrMalformed
	}
	message := &Message{
		ID:    binary.BigEndian.Uint16(data[0:2]),
		Flags: binary.BigEndian.Uint16(data[2:4]),
	}
	questionCount := int(binary.BigEndian.Uint16(data[4:6]))
	answerCount := int(binary.BigEndian.Uint16(data[6:8]))
	authorityCount := int(binary.BigEndian.Uint16(data[8:10]))
	additionalCount := int(binary.BigEndian.Uint16(data[10:12]))

	offset := 12
	var err error
	for count := 0; count < questionCount; count++ {
		question, next, questionErr := parseQuestion(data, offset)
		if questionErr != nil {
			return nil, questionErr
		}
		message.Questions = append(message.Questions, question)
		offset = next
	}
	if message.Answers, offset, err = parseRecords(data, offset, answerCount); err != nil {
		return nil, err
	}
	if message.Authorities, offset, err = parseRecords(data, offset, authorityCount); err != nil {
		return nil, err
	}
	if message.Additionals, _, err = parseRecords(data, offset, additionalCount); err != nil {
		return nil, err
	}
	return message, nil
}

func parseQuestion(data []byte, offset int) (Question, int, error) {
	name, next, err := decodeName(data, offset)
	if err != nil {
		return Question{}, 0, err
	}
	if next+4 > len(data) {
		return Question{}, 0, ErrMalformed
	}
	question := Question{
		Name:  name,
		Type:  Type(binary.BigEndian.Uint16(data[next : next+2])),
		Class: Class(binary.BigEndian.Uint16(data[next+2 : next+4])),
	}
	return question, next + 4, nil
}

func parseRecords(data []byte, offset, count int) ([]RR, int, error) {
	records := make([]RR, 0, count)
	for index := 0; index < count; index++ {
		record, next, err := parseRecord(data, offset)
		if err != nil {
			return nil, 0, err
		}
		records = append(records, record)
		offset = next
	}
	return records, offset, nil
}

func parseRecord(data []byte, offset int) (RR, int, error) {
	name, next, err := decodeName(data, offset)
	if err != nil {
		return RR{}, 0, err
	}
	if next+10 > len(data) {
		return RR{}, 0, ErrMalformed
	}
	recordType := Type(binary.BigEndian.Uint16(data[next : next+2]))
	class := Class(binary.BigEndian.Uint16(data[next+2 : next+4]))
	ttl := binary.BigEndian.Uint32(data[next+4 : next+8])
	length := int(binary.BigEndian.Uint16(data[next+8 : next+10]))
	start := next + 10
	if start+length > len(data) {
		return RR{}, 0, ErrMalformed
	}
	record := RR{Name: name, Type: recordType, Class: class, TTL: ttl, Data: data[start : start+length]}
	if err := parseRData(data, start, length, &record); err != nil {
		return RR{}, 0, err
	}
	return record, start + length, nil
}

func parseRData(data []byte, start, length int, record *RR) error {
	switch record.Type {
	case TypeA:
		if length != 4 {
			return ErrMalformed
		}
		address, ok := netip.AddrFromSlice(data[start : start+4])
		if !ok {
			return ErrMalformed
		}
		record.Address = address
	case TypeAAAA:
		if length != 16 {
			return ErrMalformed
		}
		address, ok := netip.AddrFromSlice(data[start : start+16])
		if !ok {
			return ErrMalformed
		}
		record.Address = address
	case TypeCNAME, TypeNS:
		target, _, err := decodeName(data, start)
		if err != nil {
			return err
		}
		record.Target = target
	case TypeMX:
		if length < 3 {
			return ErrMalformed
		}
		record.Priority = binary.BigEndian.Uint16(data[start : start+2])
		target, _, err := decodeName(data, start+2)
		if err != nil {
			return err
		}
		record.Target = target
	case TypeSRV:
		if length < 6 {
			return ErrMalformed
		}
		record.Priority = binary.BigEndian.Uint16(data[start : start+2])
		record.Weight = binary.BigEndian.Uint16(data[start+2 : start+4])
		record.Port = binary.BigEndian.Uint16(data[start+4 : start+6])
		target, _, err := decodeName(data, start+6)
		if err != nil {
			return err
		}
		record.Target = target
	case TypeTXT:
		position := start
		end := start + length
		for position < end {
			textLength := int(data[position])
			position++
			if position+textLength > end {
				return ErrMalformed
			}
			record.Text = append(record.Text, string(data[position:position+textLength]))
			position += textLength
		}
	}
	return nil
}

// Pack encodes a DNS message.
func (message *Message) Pack() ([]byte, error) {
	var builder []byte
	builder = append(builder, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	binary.BigEndian.PutUint16(builder[0:2], message.ID)
	binary.BigEndian.PutUint16(builder[2:4], message.Flags)
	binary.BigEndian.PutUint16(builder[4:6], uint16(len(message.Questions)))
	binary.BigEndian.PutUint16(builder[6:8], uint16(len(message.Answers)))
	binary.BigEndian.PutUint16(builder[8:10], uint16(len(message.Authorities)))
	binary.BigEndian.PutUint16(builder[10:12], uint16(len(message.Additionals)))

	var err error
	for _, question := range message.Questions {
		if builder, err = appendQuestion(builder, question); err != nil {
			return nil, err
		}
	}
	for _, record := range message.Answers {
		if builder, err = appendRecord(builder, record); err != nil {
			return nil, err
		}
	}
	for _, record := range message.Authorities {
		if builder, err = appendRecord(builder, record); err != nil {
			return nil, err
		}
	}
	for _, record := range message.Additionals {
		if builder, err = appendRecord(builder, record); err != nil {
			return nil, err
		}
	}
	return builder, nil
}

func appendQuestion(builder []byte, question Question) ([]byte, error) {
	name, err := encodeName(question.Name)
	if err != nil {
		return nil, err
	}
	builder = append(builder, name...)
	builder = binary.BigEndian.AppendUint16(builder, uint16(question.Type))
	builder = binary.BigEndian.AppendUint16(builder, uint16(question.Class))
	return builder, nil
}

func appendRecord(builder []byte, record RR) ([]byte, error) {
	name, err := encodeName(record.Name)
	if err != nil {
		return nil, err
	}
	builder = append(builder, name...)
	builder = binary.BigEndian.AppendUint16(builder, uint16(record.Type))
	builder = binary.BigEndian.AppendUint16(builder, uint16(record.Class))
	builder = binary.BigEndian.AppendUint32(builder, record.TTL)

	var data []byte
	switch record.Type {
	case TypeA, TypeAAAA:
		data = record.Address.AsSlice()
	case TypeCNAME, TypeNS:
		if data, err = encodeName(record.Target); err != nil {
			return nil, err
		}
	case TypeMX:
		data = binary.BigEndian.AppendUint16(nil, record.Priority)
		target, targetErr := encodeName(record.Target)
		if targetErr != nil {
			return nil, targetErr
		}
		data = append(data, target...)
	case TypeSRV:
		data = binary.BigEndian.AppendUint16(nil, record.Priority)
		data = binary.BigEndian.AppendUint16(data, record.Weight)
		data = binary.BigEndian.AppendUint16(data, record.Port)
		target, targetErr := encodeName(record.Target)
		if targetErr != nil {
			return nil, targetErr
		}
		data = append(data, target...)
	case TypeTXT:
		for _, text := range record.Text {
			if len(text) > 255 {
				return nil, fmt.Errorf("dns: TXT string too long")
			}
			data = append(data, byte(len(text)))
			data = append(data, text...)
		}
	default:
		data = record.Data
	}
	builder = binary.BigEndian.AppendUint16(builder, uint16(len(data)))
	builder = append(builder, data...)
	return builder, nil
}

func decodeName(message []byte, offset int) (string, int, error) {
	var labels []string
	position := offset
	next := -1
	jumps := 0
	for {
		if position >= len(message) {
			return "", 0, ErrMalformed
		}
		length := int(message[position])
		if length == 0 {
			position++
			if next == -1 {
				next = position
			}
			break
		}
		if length&0xC0 == 0xC0 {
			if position+1 >= len(message) {
				return "", 0, ErrMalformed
			}
			pointer := int(message[position]&0x3F)<<8 | int(message[position+1])
			if next == -1 {
				next = position + 2
			}
			jumps++
			if jumps > 64 || pointer >= len(message) {
				return "", 0, ErrMalformed
			}
			position = pointer
			continue
		}
		if length > 63 || position+1+length > len(message) {
			return "", 0, ErrMalformed
		}
		labels = append(labels, string(message[position+1:position+1+length]))
		position += 1 + length
	}
	return strings.Join(labels, "."), next, nil
}

func encodeName(name string) ([]byte, error) {
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return []byte{0}, nil
	}
	var builder []byte
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 {
			return nil, fmt.Errorf("dns: invalid label %q", label)
		}
		builder = append(builder, byte(len(label)))
		builder = append(builder, label...)
	}
	return append(builder, 0), nil
}
