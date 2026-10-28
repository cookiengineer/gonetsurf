package x11

import "encoding/binary"

func createWindowRequest(depth uint8, windowID, parent uint32, width, height uint16, visual uint32, valueMask uint32, values []uint32) []byte {
	length := 32 + 4*len(values)
	request := make([]byte, length)
	request[0] = opcodeCreateWindow
	request[1] = depth
	binary.LittleEndian.PutUint16(request[2:4], uint16(length/4))
	binary.LittleEndian.PutUint32(request[4:8], windowID)
	binary.LittleEndian.PutUint32(request[8:12], parent)
	binary.LittleEndian.PutUint16(request[16:18], width)
	binary.LittleEndian.PutUint16(request[18:20], height)
	binary.LittleEndian.PutUint16(request[22:24], windowClassInputOutput)
	binary.LittleEndian.PutUint32(request[24:28], visual)
	binary.LittleEndian.PutUint32(request[28:32], valueMask)
	for index, value := range values {
		binary.LittleEndian.PutUint32(request[32+index*4:], value)
	}
	return request
}

func changePropertyRequest(window, property, propertyType uint32, format uint8, data []byte) []byte {
	units := len(data)
	switch format {
	case 16:
		units = len(data) / 2
	case 32:
		units = len(data) / 4
	}
	paddedLength := len(data) + pad4(len(data))
	length := 24 + paddedLength
	request := make([]byte, length)
	request[0] = opcodeChangeProperty
	request[1] = propertyModeReplace
	binary.LittleEndian.PutUint16(request[2:4], uint16(length/4))
	binary.LittleEndian.PutUint32(request[4:8], window)
	binary.LittleEndian.PutUint32(request[8:12], property)
	binary.LittleEndian.PutUint32(request[12:16], propertyType)
	request[16] = format
	binary.LittleEndian.PutUint32(request[20:24], uint32(units))
	copy(request[24:], data)
	return request
}

func internAtomRequest(name string) []byte {
	nameBytes := []byte(name)
	paddedLength := len(nameBytes) + pad4(len(nameBytes))
	length := 8 + paddedLength
	request := make([]byte, length)
	request[0] = opcodeInternAtom
	binary.LittleEndian.PutUint16(request[2:4], uint16(length/4))
	binary.LittleEndian.PutUint16(request[4:6], uint16(len(nameBytes)))
	copy(request[8:], nameBytes)
	return request
}

func mapWindowRequest(window uint32) []byte {
	request := make([]byte, 8)
	request[0] = opcodeMapWindow
	binary.LittleEndian.PutUint16(request[2:4], 2)
	binary.LittleEndian.PutUint32(request[4:8], window)
	return request
}

func destroyWindowRequest(window uint32) []byte {
	request := make([]byte, 8)
	request[0] = opcodeDestroyWindow
	binary.LittleEndian.PutUint16(request[2:4], 2)
	binary.LittleEndian.PutUint32(request[4:8], window)
	return request
}

func freeGCRequest(gc uint32) []byte {
	request := make([]byte, 8)
	request[0] = opcodeFreeGC
	binary.LittleEndian.PutUint16(request[2:4], 2)
	binary.LittleEndian.PutUint32(request[4:8], gc)
	return request
}

func createGCRequest(gc, drawable uint32, valueMask uint32, values []uint32) []byte {
	length := 16 + 4*len(values)
	request := make([]byte, length)
	request[0] = opcodeCreateGC
	binary.LittleEndian.PutUint16(request[2:4], uint16(length/4))
	binary.LittleEndian.PutUint32(request[4:8], gc)
	binary.LittleEndian.PutUint32(request[8:12], drawable)
	binary.LittleEndian.PutUint32(request[12:16], valueMask)
	for index, value := range values {
		binary.LittleEndian.PutUint32(request[16+index*4:], value)
	}
	return request
}

func putImageRequest(drawable, gc uint32, width, height uint16, destinationX, destinationY int16, depth uint8, data []byte) []byte {
	paddedLength := len(data) + pad4(len(data))
	length := 24 + paddedLength
	request := make([]byte, length)
	request[0] = opcodePutImage
	request[1] = imageFormatZPixmap
	binary.LittleEndian.PutUint16(request[2:4], uint16(length/4))
	binary.LittleEndian.PutUint32(request[4:8], drawable)
	binary.LittleEndian.PutUint32(request[8:12], gc)
	binary.LittleEndian.PutUint16(request[12:14], width)
	binary.LittleEndian.PutUint16(request[14:16], height)
	binary.LittleEndian.PutUint16(request[16:18], uint16(destinationX))
	binary.LittleEndian.PutUint16(request[18:20], uint16(destinationY))
	request[21] = depth
	copy(request[24:], data)
	return request
}
