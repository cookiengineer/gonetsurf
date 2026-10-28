package fonts

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

type woffTable struct {
	tag        uint32
	offset     uint32
	compLength uint32
	origLength uint32
	checksum   uint32
}

func decodeWOFF(data []byte) ([]byte, error) {
	if len(data) < 44 || string(data[:4]) != "wOFF" {
		return nil, fmt.Errorf("fonts: invalid WOFF header")
	}
	flavor := binary.BigEndian.Uint32(data[4:8])
	numTables := int(binary.BigEndian.Uint16(data[12:14]))
	if numTables <= 0 || numTables > 4096 {
		return nil, fmt.Errorf("fonts: invalid WOFF table count")
	}
	if 44+numTables*20 > len(data) {
		return nil, fmt.Errorf("fonts: truncated WOFF directory")
	}
	tables := make([]woffTable, numTables)
	for index := 0; index < numTables; index++ {
		offset := 44 + index*20
		tables[index] = woffTable{
			tag:        binary.BigEndian.Uint32(data[offset : offset+4]),
			offset:     binary.BigEndian.Uint32(data[offset+4 : offset+8]),
			compLength: binary.BigEndian.Uint32(data[offset+8 : offset+12]),
			origLength: binary.BigEndian.Uint32(data[offset+12 : offset+16]),
			checksum:   binary.BigEndian.Uint32(data[offset+16 : offset+20]),
		}
		if tables[index].origLength > 1<<28 {
			return nil, fmt.Errorf("fonts: WOFF table is too large")
		}
		if uint64(tables[index].offset)+uint64(tables[index].compLength) > uint64(len(data)) {
			return nil, fmt.Errorf("fonts: truncated WOFF table data")
		}
	}

	entrySelector := 0
	for (1 << (entrySelector + 1)) <= numTables {
		entrySelector++
	}
	searchRange := (1 << entrySelector) * 16
	rangeShift := numTables*16 - searchRange

	headerLength := 12 + numTables*16
	output := make([]byte, headerLength)
	binary.BigEndian.PutUint32(output[0:4], flavor)
	binary.BigEndian.PutUint16(output[4:6], uint16(numTables))
	binary.BigEndian.PutUint16(output[6:8], uint16(searchRange))
	binary.BigEndian.PutUint16(output[8:10], uint16(entrySelector))
	binary.BigEndian.PutUint16(output[10:12], uint16(rangeShift))

	tableDataOffset := headerLength
	for index, table := range tables {
		offset := 12 + index*16
		binary.BigEndian.PutUint32(output[offset:offset+4], table.tag)
		binary.BigEndian.PutUint32(output[offset+4:offset+8], table.checksum)
		binary.BigEndian.PutUint32(output[offset+8:offset+12], uint32(tableDataOffset))
		binary.BigEndian.PutUint32(output[offset+12:offset+16], table.origLength)

		compressed := data[table.offset : table.offset+table.compLength]
		var tableData []byte
		if table.compLength < table.origLength {
			reader, err := zlib.NewReader(bytes.NewReader(compressed))
			if err != nil {
				return nil, fmt.Errorf("fonts: WOFF decompression: %w", err)
			}
			decompressed, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				return nil, fmt.Errorf("fonts: WOFF decompression: %w", err)
			}
			if len(decompressed) != int(table.origLength) {
				return nil, fmt.Errorf("fonts: WOFF table length mismatch")
			}
			tableData = decompressed
		} else {
			if table.compLength != table.origLength {
				return nil, fmt.Errorf("fonts: invalid WOFF table lengths")
			}
			tableData = compressed
		}
		output = append(output, tableData...)
		for len(output)%4 != 0 {
			output = append(output, 0)
		}
		tableDataOffset += len(tableData)
		tableDataOffset += (4 - tableDataOffset%4) % 4
	}
	return output, nil
}
