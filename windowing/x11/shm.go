package x11

import (
	"encoding/binary"
	"image"
)

const maxSHMBytes = 64 << 20

type shmSegment struct {
	segmentID uint32
	systemID  int
	data      []byte
	width     int
	height    int
}

func (display *Display) querySHM() {
	sequence, err := display.sendRequest(queryExtensionRequest("MIT-SHM"))
	if err != nil {
		return
	}
	reply, err := display.waitForReplyData(sequence)
	if err != nil || len(reply) < 12 {
		return
	}
	if reply[8] != 0 {
		display.shmSupported = true
		display.shmMajor = reply[9]
	}
}

func (display *Display) enableSHM(width, height int) bool {
	if !display.shmSupported || display.shmMajor == 0 {
		return false
	}
	display.shmMutex.Lock()
	defer display.shmMutex.Unlock()
	display.disableSHMLocked()
	size := width * height * 4
	if width <= 0 || height <= 0 || size <= 0 || size > maxSHMBytes {
		return false
	}
	systemID, data, err := createShmSegment(size)
	if err != nil {
		return false
	}
	segmentID := display.allocateID()
	if _, err := display.sendRequest(shmAttachRequest(display.shmMajor, segmentID, uint32(systemID))); err != nil {
		destroyShmSegment(systemID, data)
		return false
	}
	display.shm = &shmSegment{segmentID: segmentID, systemID: systemID, data: data, width: width, height: height}
	return true
}

func (display *Display) disableSHM() {
	display.shmMutex.Lock()
	defer display.shmMutex.Unlock()
	display.disableSHMLocked()
}

func (display *Display) disableSHMLocked() {
	if display.shm == nil {
		return
	}
	display.sendRequest(shmDetachRequest(display.shmMajor, display.shm.segmentID))
	destroyShmSegment(display.shm.systemID, display.shm.data)
	display.shm = nil
}

func (display *Display) resizeSHM(width, height int) bool {
	if !display.shmSupported {
		return false
	}
	return display.enableSHM(width, height)
}

func (window *Window) presentSHM(converted *image.NRGBA, width, height, startRow, endRow int) bool {
	display := window.display
	display.shmMutex.Lock()
	defer display.shmMutex.Unlock()
	segment := display.shm
	if segment == nil || segment.width != width || segment.height != height {
		return false
	}
	stride := width * 4
	offset := startRow * stride
	bandHeight := endRow - startRow
	if bandHeight <= 0 || offset+bandHeight*stride > len(segment.data) {
		return false
	}
	data := segment.data[offset : offset+bandHeight*stride]
	encodeNRGBABand(converted, width, startRow, endRow, display.setup.imageByteOrder, [3]uint32{
		display.visual.redMask,
		display.visual.greenMask,
		display.visual.blueMask,
	}, data)
	request := shmPutImageRequest(
		display.shmMajor,
		window.id,
		window.gc,
		segment.segmentID,
		uint32(offset),
		uint16(width),
		uint16(height),
		0,
		0,
		uint16(width),
		uint16(bandHeight),
		0,
		int16(startRow),
		display.visual.depth,
	)
	if _, err := display.sendRequest(request); err != nil {
		return false
	}
	return true
}

func encodeNRGBABand(converted *image.NRGBA, width, startRow, endRow int, byteOrder uint8, masks [3]uint32, destination []byte) {
	offset := 0
	for row := startRow; row < endRow; row++ {
		for column := 0; column < width; column++ {
			sourceOffset := row*converted.Stride + column*4
			value := packPixel(
				converted.Pix[sourceOffset],
				converted.Pix[sourceOffset+1],
				converted.Pix[sourceOffset+2],
				masks,
			)
			if byteOrder == 0 {
				binary.LittleEndian.PutUint32(destination[offset:], value)
			} else {
				binary.BigEndian.PutUint32(destination[offset:], value)
			}
			offset += 4
		}
	}
}
