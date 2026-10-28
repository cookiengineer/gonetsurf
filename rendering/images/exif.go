package images

import (
	"encoding/binary"
	"image"
)

func exifOrientation(data []byte) Orientation {
	orientation := Orientation(1)
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return orientation
	}
	offset := 2
	for offset+4 <= len(data) {
		for offset < len(data) && data[offset] == 0xFF {
			offset++
		}
		if offset >= len(data) {
			break
		}
		marker := data[offset]
		offset++
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}
		if marker == 0xD9 || marker == 0xDA {
			break
		}
		if offset+2 > len(data) {
			break
		}
		length := int(binary.BigEndian.Uint16(data[offset : offset+2]))
		if length < 2 || offset+length > len(data) {
			break
		}
		segment := data[offset+2 : offset+length]
		if marker == 0xE1 && len(segment) > 6 && string(segment[:6]) == "Exif\x00\x00" {
			if parsed := parseTIFFOrientation(segment[6:]); parsed >= 1 && parsed <= 8 {
				return Orientation(parsed)
			}
		}
		offset += length
	}
	return orientation
}

func parseTIFFOrientation(data []byte) int {
	if len(data) < 8 {
		return 0
	}
	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	if order.Uint16(data[2:4]) != 42 {
		return 0
	}
	ifdOffset := int(order.Uint32(data[4:8]))
	if ifdOffset < 0 || ifdOffset+2 > len(data) {
		return 0
	}
	count := int(order.Uint16(data[ifdOffset : ifdOffset+2]))
	offset := ifdOffset + 2
	for entry := 0; entry < count; entry++ {
		if offset+12 > len(data) {
			return 0
		}
		tag := order.Uint16(data[offset : offset+2])
		fieldType := order.Uint16(data[offset+2 : offset+4])
		valueCount := order.Uint32(data[offset+4 : offset+8])
		if tag == 0x0112 && valueCount >= 1 {
			switch fieldType {
			case 3:
				return int(order.Uint16(data[offset+8 : offset+10]))
			case 4:
				valueOffset := int(order.Uint32(data[offset+8 : offset+12]))
				if valueOffset+2 <= len(data) {
					return int(order.Uint16(data[valueOffset : valueOffset+2]))
				}
			}
		}
		offset += 12
	}
	return 0
}

func applyOrientation(source *image.NRGBA, orientation Orientation) *image.NRGBA {
	sourceWidth := source.Bounds().Dx()
	sourceHeight := source.Bounds().Dy()
	swapped := orientation >= 5 && orientation <= 8
	destinationWidth := sourceWidth
	destinationHeight := sourceHeight
	if swapped {
		destinationWidth, destinationHeight = sourceHeight, sourceWidth
	}
	destination := image.NewNRGBA(image.Rect(0, 0, destinationWidth, destinationHeight))
	for yPos := 0; yPos < destinationHeight; yPos++ {
		for xPos := 0; xPos < destinationWidth; xPos++ {
			sourceX, sourceY := xPos, yPos
			switch orientation {
			case 2:
				sourceX = sourceWidth - 1 - xPos
			case 3:
				sourceX = sourceWidth - 1 - xPos
				sourceY = sourceHeight - 1 - yPos
			case 4:
				sourceY = sourceHeight - 1 - yPos
			case 5:
				sourceX, sourceY = yPos, xPos
			case 6:
				sourceX, sourceY = yPos, sourceHeight-1-xPos
			case 7:
				sourceX, sourceY = sourceWidth-1-yPos, sourceHeight-1-xPos
			case 8:
				sourceX, sourceY = sourceWidth-1-yPos, xPos
			}
			if sourceX < 0 || sourceY < 0 || sourceX >= sourceWidth || sourceY >= sourceHeight {
				continue
			}
			sourceOffset := sourceY*source.Stride + sourceX*4
			destinationOffset := yPos*destination.Stride + xPos*4
			copy(destination.Pix[destinationOffset:destinationOffset+4], source.Pix[sourceOffset:sourceOffset+4])
		}
	}
	return destination
}
