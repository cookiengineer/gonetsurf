package x11

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"net"
	"os"
	"strconv"
	"strings"
)

// X11 request opcodes used by the client.
const (
	opcodeCreateWindow     = 1
	opcodeDestroyWindow    = 4
	opcodeMapWindow        = 8
	opcodeInternAtom       = 16
	opcodeChangeProperty   = 18
	opcodeCreateGC         = 55
	opcodeFreeGC           = 60
	opcodePutImage         = 72
	opcodeGetInputFocus    = 43
	propertyModeReplace    = 0
	imageFormatZPixmap     = 2
	windowClassInputOutput = 1
	visualClassTrueColor   = 4
)

const (
	eventMaskKeyPress        = 1 << 0
	eventMaskKeyRelease      = 1 << 1
	eventMaskButtonPress     = 1 << 2
	eventMaskButtonRelease   = 1 << 3
	eventMaskPointerMotion   = 1 << 6
	eventMaskButtonMotion    = 1 << 13
	eventMaskExposure        = 1 << 15
	eventMaskStructureNotify = 1 << 17
	eventMaskFocusChange     = 1 << 21
	eventMaskPropertyChange  = 1 << 22
)

const (
	eventExpose           = 12
	eventDestroyNotify    = 17
	eventConfigureNotify  = 22
	eventSelectionClear   = 29
	eventSelectionRequest = 30
	eventSelectionNotify  = 31
	eventClientMessage    = 33
	eventGeneric          = 35
)

const (
	atomString      = 31
	atomWMName      = 39
	familyInternet  = 0
	familyInternet6 = 6
	familyLocal     = 256
	familyWild      = 65535
	magicCookieName = "MIT-MAGIC-COOKIE-1"
	protocolMajor   = 11
	protocolMinor   = 0
)

var errorUnsupported = errors.New("x11: unsupported server")

func pad4(length int) int {
	return (4 - length%4) % 4
}

type displayName struct {
	host   string
	number int
	screen int
	isUnix bool
}

func parseDisplayName(value string) (displayName, error) {
	if value == "" {
		return displayName{}, fmt.Errorf("x11: empty display name")
	}
	colon := strings.LastIndexByte(value, ':')
	if colon < 0 {
		return displayName{}, fmt.Errorf("x11: invalid display name %q", value)
	}
	host := value[:colon]
	rest := value[colon+1:]
	numberPart := rest
	screenPart := "0"
	if dot := strings.IndexByte(rest, '.'); dot >= 0 {
		numberPart = rest[:dot]
		screenPart = rest[dot+1:]
	}
	number, err := strconv.Atoi(numberPart)
	if err != nil || number < 0 {
		return displayName{}, fmt.Errorf("x11: invalid display number %q", rest)
	}
	screen, err := strconv.Atoi(screenPart)
	if err != nil || screen < 0 {
		return displayName{}, fmt.Errorf("x11: invalid screen number %q", screenPart)
	}
	isUnix := host == "" || host == "unix" || host == "localhost"
	return displayName{host: host, number: number, screen: screen, isUnix: isUnix}, nil
}

func (name displayName) address() string {
	if name.isUnix {
		return "/tmp/.X11-unix/X" + strconv.Itoa(name.number)
	}
	return net.JoinHostPort(name.host, strconv.Itoa(6000+name.number))
}

type authorityEntry struct {
	family  uint16
	address []byte
	number  []byte
	name    []byte
	data    []byte
}

func readAuthorityFile(path string) ([]authorityEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	var entries []authorityEntry
	reader := bufio.NewReader(file)
	for {
		entry, err := readAuthorityEntry(reader)
		if err == io.EOF {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
}

func readAuthorityEntry(reader *bufio.Reader) (authorityEntry, error) {
	var entry authorityEntry
	var header [2]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return entry, err
	}
	entry.family = binary.BigEndian.Uint16(header[:])
	fields := []*[]byte{&entry.address, &entry.number, &entry.name, &entry.data}
	for _, field := range fields {
		if _, err := io.ReadFull(reader, header[:]); err != nil {
			return entry, err
		}
		length := int(binary.BigEndian.Uint16(header[:]))
		value := make([]byte, length)
		if _, err := io.ReadFull(reader, value); err != nil {
			return entry, err
		}
		*field = value
	}
	return entry, nil
}

func authorityFor(name displayName, hostname string) ([]byte, []byte) {
	path := os.Getenv("XAUTHORITY")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, nil
		}
		path = home + "/.Xauthority"
	}
	entries, err := readAuthorityFile(path)
	if err != nil {
		return nil, nil
	}
	number := []byte(strconv.Itoa(name.number))
	for _, entry := range entries {
		if string(entry.name) != magicCookieName || string(entry.number) != string(number) {
			continue
		}
		if authorityMatches(entry, name, hostname) {
			return entry.name, entry.data
		}
	}
	return nil, nil
}

func authorityMatches(entry authorityEntry, name displayName, hostname string) bool {
	switch entry.family {
	case familyWild:
		return true
	case familyLocal:
		return name.isUnix && string(entry.address) == hostname
	case familyInternet:
		if name.isUnix {
			return false
		}
		return string(entry.address) == name.host
	case familyInternet6:
		if name.isUnix {
			return false
		}
		return string(entry.address) == name.host
	default:
		return false
	}
}

func setupRequest(authName, authData []byte) []byte {
	nameLength := len(authName)
	dataLength := len(authData)
	total := 12 + nameLength + pad4(nameLength) + dataLength + pad4(dataLength)
	request := make([]byte, total)
	request[0] = 'l'
	binary.LittleEndian.PutUint16(request[2:4], protocolMajor)
	binary.LittleEndian.PutUint16(request[4:6], protocolMinor)
	binary.LittleEndian.PutUint16(request[6:8], uint16(nameLength))
	binary.LittleEndian.PutUint16(request[8:10], uint16(dataLength))
	copy(request[12:], authName)
	copy(request[12+nameLength+pad4(nameLength):], authData)
	return request
}

type pixmapFormat struct {
	depth        uint8
	bitsPerPixel uint8
	scanlinePad  uint8
}

type visualInfo struct {
	id              uint32
	class           uint8
	bitsPerRGB      uint8
	colormapEntries uint16
	redMask         uint32
	greenMask       uint32
	blueMask        uint32
	depth           uint8
}

type screenInfo struct {
	root       uint32
	whitePixel uint32
	blackPixel uint32
	width      uint16
	height     uint16
	rootVisual uint32
	rootDepth  uint8
	visuals    []visualInfo
}

type setupInfo struct {
	resourceBase     uint32
	resourceMask     uint32
	imageByteOrder   uint8
	maxRequestLength uint16
	minKeycode       uint8
	maxKeycode       uint8
	screens          []screenInfo
	formats          []pixmapFormat
}

func readSetup(reader io.Reader) (*setupInfo, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}
	status := header[0]
	reasonLength := int(header[1])
	length := int(binary.LittleEndian.Uint16(header[6:8])) * 4
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	if status != 1 {
		reason := ""
		if reasonLength > 0 && reasonLength <= len(body) {
			reason = string(body[:reasonLength])
		}
		return nil, fmt.Errorf("x11: connection setup failed: %s", reason)
	}
	if len(body) < 32 {
		return nil, fmt.Errorf("x11: truncated setup response")
	}
	info := &setupInfo{
		resourceBase:     binary.LittleEndian.Uint32(body[4:8]),
		resourceMask:     binary.LittleEndian.Uint32(body[8:12]),
		imageByteOrder:   body[22],
		maxRequestLength: binary.LittleEndian.Uint16(body[18:20]),
		minKeycode:       body[26],
		maxKeycode:       body[27],
	}
	vendorLength := int(binary.LittleEndian.Uint16(body[16:18]))
	screenCount := int(body[20])
	formatCount := int(body[21])
	offset := 32
	if offset+vendorLength > len(body) {
		return nil, fmt.Errorf("x11: truncated vendor string")
	}
	offset += vendorLength + pad4(vendorLength)

	for index := 0; index < formatCount; index++ {
		if offset+8 > len(body) {
			return nil, fmt.Errorf("x11: truncated pixmap format")
		}
		info.formats = append(info.formats, pixmapFormat{
			depth:        body[offset],
			bitsPerPixel: body[offset+1],
			scanlinePad:  body[offset+2],
		})
		offset += 8
	}

	for index := 0; index < screenCount; index++ {
		if offset+40 > len(body) {
			return nil, fmt.Errorf("x11: truncated screen")
		}
		screen := screenInfo{
			root:       binary.LittleEndian.Uint32(body[offset : offset+4]),
			whitePixel: binary.LittleEndian.Uint32(body[offset+8 : offset+12]),
			blackPixel: binary.LittleEndian.Uint32(body[offset+12 : offset+16]),
			width:      binary.LittleEndian.Uint16(body[offset+20 : offset+22]),
			height:     binary.LittleEndian.Uint16(body[offset+22 : offset+24]),
			rootVisual: binary.LittleEndian.Uint32(body[offset+32 : offset+36]),
			rootDepth:  body[offset+38],
		}
		depthCount := int(body[offset+39])
		offset += 40
		for depthIndex := 0; depthIndex < depthCount; depthIndex++ {
			if offset+8 > len(body) {
				return nil, fmt.Errorf("x11: truncated depth")
			}
			depth := body[offset]
			visualCount := int(binary.LittleEndian.Uint16(body[offset+2 : offset+4]))
			offset += 8
			for visualIndex := 0; visualIndex < visualCount; visualIndex++ {
				if offset+24 > len(body) {
					return nil, fmt.Errorf("x11: truncated visual")
				}
				screen.visuals = append(screen.visuals, visualInfo{
					id:              binary.LittleEndian.Uint32(body[offset : offset+4]),
					class:           body[offset+4],
					bitsPerRGB:      body[offset+5],
					colormapEntries: binary.LittleEndian.Uint16(body[offset+6 : offset+8]),
					redMask:         binary.LittleEndian.Uint32(body[offset+8 : offset+12]),
					greenMask:       binary.LittleEndian.Uint32(body[offset+12 : offset+16]),
					blueMask:        binary.LittleEndian.Uint32(body[offset+16 : offset+20]),
					depth:           depth,
				})
				offset += 24
			}
		}
		info.screens = append(info.screens, screen)
	}
	return info, nil
}

func (info *setupInfo) selectVisual(screen screenInfo) (visualInfo, int, error) {
	var candidates []visualInfo
	for _, visual := range screen.visuals {
		if visual.id == screen.rootVisual && visual.class == visualClassTrueColor {
			candidates = append(candidates, visual)
		}
	}
	for _, visual := range screen.visuals {
		if visual.class == visualClassTrueColor && visual.depth == screen.rootDepth {
			candidates = append(candidates, visual)
		}
	}
	for _, visual := range candidates {
		for _, format := range info.formats {
			if format.depth == visual.depth && format.bitsPerPixel == 32 {
				return visual, 32, nil
			}
		}
	}
	return visualInfo{}, 0, errorUnsupported
}

func packPixel(red, green, blue byte, masks [3]uint32) uint32 {
	return packChannel(red, masks[0]) |
		packChannel(green, masks[1]) |
		packChannel(blue, masks[2])
}

func packChannel(component byte, mask uint32) uint32 {
	if mask == 0 {
		return 0
	}
	shift := bits.TrailingZeros32(mask)
	width := bits.OnesCount32(mask)
	maximum := uint32(1)<<width - 1
	value := uint32(component) * maximum / 255
	return (value << shift) & mask
}
