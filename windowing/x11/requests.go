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

func getKeyboardMappingRequest(firstKeycode, count uint8) []byte {
	request := make([]byte, 8)
	request[0] = 101
	binary.LittleEndian.PutUint16(request[2:4], 2)
	request[4] = firstKeycode
	request[5] = count
	return request
}

func getPropertyRequest(window, property, propertyType uint32, offset, length uint32) []byte {
	request := make([]byte, 24)
	request[0] = 20
	binary.LittleEndian.PutUint16(request[2:4], 6)
	binary.LittleEndian.PutUint32(request[4:8], window)
	binary.LittleEndian.PutUint32(request[8:12], property)
	binary.LittleEndian.PutUint32(request[12:16], propertyType)
	binary.LittleEndian.PutUint32(request[16:20], offset)
	binary.LittleEndian.PutUint32(request[20:24], length)
	return request
}

func setSelectionOwnerRequest(owner, selection, timestamp uint32) []byte {
	request := make([]byte, 16)
	request[0] = 22
	binary.LittleEndian.PutUint16(request[2:4], 4)
	binary.LittleEndian.PutUint32(request[4:8], owner)
	binary.LittleEndian.PutUint32(request[8:12], selection)
	binary.LittleEndian.PutUint32(request[12:16], timestamp)
	return request
}

func getSelectionOwnerRequest(selection uint32) []byte {
	request := make([]byte, 8)
	request[0] = 23
	binary.LittleEndian.PutUint16(request[2:4], 2)
	binary.LittleEndian.PutUint32(request[4:8], selection)
	return request
}

func convertSelectionRequest(requestor, selection, target, property, timestamp uint32) []byte {
	request := make([]byte, 24)
	request[0] = 24
	binary.LittleEndian.PutUint16(request[2:4], 6)
	binary.LittleEndian.PutUint32(request[4:8], requestor)
	binary.LittleEndian.PutUint32(request[8:12], selection)
	binary.LittleEndian.PutUint32(request[12:16], target)
	binary.LittleEndian.PutUint32(request[16:20], property)
	binary.LittleEndian.PutUint32(request[20:24], timestamp)
	return request
}

func sendEventRequest(destination, eventMask uint32, event []byte) []byte {
	request := make([]byte, 44)
	request[0] = 25
	binary.LittleEndian.PutUint16(request[2:4], 11)
	binary.LittleEndian.PutUint32(request[4:8], destination)
	binary.LittleEndian.PutUint32(request[8:12], eventMask)
	copy(request[12:44], event)
	return request
}

func deletePropertyRequest(window, property uint32) []byte {
	request := make([]byte, 12)
	request[0] = 19
	binary.LittleEndian.PutUint16(request[2:4], 3)
	binary.LittleEndian.PutUint32(request[4:8], window)
	binary.LittleEndian.PutUint32(request[8:12], property)
	return request
}

func selectionNotifyEvent(timestamp, requestor, selection, target, property uint32) []byte {
	event := make([]byte, 32)
	event[0] = 31
	binary.LittleEndian.PutUint32(event[4:8], timestamp)
	binary.LittleEndian.PutUint32(event[8:12], requestor)
	binary.LittleEndian.PutUint32(event[12:16], selection)
	binary.LittleEndian.PutUint32(event[16:20], target)
	binary.LittleEndian.PutUint32(event[20:24], property)
	return event
}

func queryExtensionRequest(name string) []byte {
	nameBytes := []byte(name)
	paddedLength := len(nameBytes) + pad4(len(nameBytes))
	request := make([]byte, 8+paddedLength)
	request[0] = 98
	binary.LittleEndian.PutUint16(request[2:4], uint16((8+paddedLength)/4))
	binary.LittleEndian.PutUint16(request[4:6], uint16(len(nameBytes)))
	copy(request[8:], nameBytes)
	return request
}

func shmAttachRequest(majorOpcode byte, segmentID, systemID uint32) []byte {
	request := make([]byte, 16)
	request[0] = majorOpcode
	request[1] = 1
	binary.LittleEndian.PutUint16(request[2:4], 4)
	binary.LittleEndian.PutUint32(request[4:8], segmentID)
	binary.LittleEndian.PutUint32(request[8:12], systemID)
	return request
}

func shmDetachRequest(majorOpcode byte, segmentID uint32) []byte {
	request := make([]byte, 12)
	request[0] = majorOpcode
	request[1] = 2
	binary.LittleEndian.PutUint16(request[2:4], 3)
	binary.LittleEndian.PutUint32(request[4:8], segmentID)
	return request
}

func shmPutImageRequest(majorOpcode byte, drawable, gc, segmentID, offset uint32, totalWidth, totalHeight uint16, sourceX, sourceY int16, sourceWidth, sourceHeight uint16, destinationX, destinationY int16, depth uint8) []byte {
	request := make([]byte, 40)
	request[0] = majorOpcode
	request[1] = 3
	binary.LittleEndian.PutUint16(request[2:4], 10)
	binary.LittleEndian.PutUint32(request[4:8], drawable)
	binary.LittleEndian.PutUint32(request[8:12], gc)
	binary.LittleEndian.PutUint16(request[12:14], totalWidth)
	binary.LittleEndian.PutUint16(request[14:16], totalHeight)
	binary.LittleEndian.PutUint16(request[16:18], uint16(sourceX))
	binary.LittleEndian.PutUint16(request[18:20], uint16(sourceY))
	binary.LittleEndian.PutUint16(request[20:22], sourceWidth)
	binary.LittleEndian.PutUint16(request[22:24], sourceHeight)
	binary.LittleEndian.PutUint16(request[24:26], uint16(destinationX))
	binary.LittleEndian.PutUint16(request[26:28], uint16(destinationY))
	request[28] = depth
	request[29] = imageFormatZPixmap
	binary.LittleEndian.PutUint32(request[32:36], segmentID)
	binary.LittleEndian.PutUint32(request[36:40], offset)
	return request
}

func getImageRequest(drawable uint32, xPos, yPos int16, width, height uint16) []byte {
	request := make([]byte, 20)
	request[0] = 73
	request[1] = imageFormatZPixmap
	binary.LittleEndian.PutUint16(request[2:4], 5)
	binary.LittleEndian.PutUint32(request[4:8], drawable)
	binary.LittleEndian.PutUint16(request[8:10], uint16(xPos))
	binary.LittleEndian.PutUint16(request[10:12], uint16(yPos))
	binary.LittleEndian.PutUint16(request[12:14], width)
	binary.LittleEndian.PutUint16(request[14:16], height)
	binary.LittleEndian.PutUint32(request[16:20], 0xffffffff)
	return request
}
