package rendering

// CompositeOp is a Porter-Duff compositing operator.
type CompositeOp uint8

const (
	// SourceOver places the source over the destination.
	SourceOver CompositeOp = iota
	// SourceIn keeps the source inside the destination alpha.
	SourceIn
	// SourceOut keeps the source outside the destination alpha.
	SourceOut
	// SourceAtop places the source over the destination, clipped to it.
	SourceAtop
	// DestinationOver places the destination over the source.
	DestinationOver
	// DestinationIn keeps the destination inside the source alpha.
	DestinationIn
	// DestinationOut keeps the destination outside the source alpha.
	DestinationOut
	// DestinationAtop places the destination over the source, clipped to it.
	DestinationAtop
	// Xor keeps the non-overlapping parts of both images.
	Xor
	// Clear clears the destination.
	Clear
	// Copy replaces the destination with the source.
	Copy
)

// Blit composites source onto the surface at the given offset.
func (surface *Surface) Blit(offsetX, offsetY int, source *Surface, op CompositeOp) {
	if source == nil {
		return
	}
	for sourceY := 0; sourceY < source.Height; sourceY++ {
		destinationY := offsetY + sourceY
		if destinationY < 0 || destinationY >= surface.Height {
			continue
		}
		for sourceX := 0; sourceX < source.Width; sourceX++ {
			destinationX := offsetX + sourceX
			if destinationX < 0 || destinationX >= surface.Width {
				continue
			}
			if !pointInRect(float64(destinationX)+0.5, float64(destinationY)+0.5, surface.clip) {
				continue
			}
			sourceOffset := sourceY*source.Stride + sourceX*4
			destinationOffset := destinationY*surface.Stride + destinationX*4
			compositePixel(surface.Pix, destinationOffset, source.Pix, sourceOffset, op)
		}
	}
}

func compositePixel(destination []uint8, destinationOffset int, source []uint8, sourceOffset int, op CompositeOp) {
	sourceAlpha := float64(source[sourceOffset+3]) / 255
	destinationAlpha := float64(destination[destinationOffset+3]) / 255
	sourceFactor, destinationFactor := compositeFactors(op, sourceAlpha, destinationAlpha)

	sourceRed := float64(source[sourceOffset]) / 255
	sourceGreen := float64(source[sourceOffset+1]) / 255
	sourceBlue := float64(source[sourceOffset+2]) / 255
	destinationRed := float64(destination[destinationOffset]) / 255
	destinationGreen := float64(destination[destinationOffset+1]) / 255
	destinationBlue := float64(destination[destinationOffset+2]) / 255

	destination[destinationOffset] = toByte(sourceRed*sourceFactor + destinationRed*destinationFactor)
	destination[destinationOffset+1] = toByte(sourceGreen*sourceFactor + destinationGreen*destinationFactor)
	destination[destinationOffset+2] = toByte(sourceBlue*sourceFactor + destinationBlue*destinationFactor)
	destination[destinationOffset+3] = toByte(sourceAlpha*sourceFactor + destinationAlpha*destinationFactor)
}

func compositeFactors(op CompositeOp, sourceAlpha, destinationAlpha float64) (sourceFactor, destinationFactor float64) {
	switch op {
	case SourceOver:
		return 1, 1 - sourceAlpha
	case SourceIn:
		return destinationAlpha, 0
	case SourceOut:
		return 1 - destinationAlpha, 0
	case SourceAtop:
		return destinationAlpha, 1 - sourceAlpha
	case DestinationOver:
		return 1 - destinationAlpha, 1
	case DestinationIn:
		return 0, sourceAlpha
	case DestinationOut:
		return 0, 1 - sourceAlpha
	case DestinationAtop:
		return 1 - destinationAlpha, sourceAlpha
	case Xor:
		return 1 - destinationAlpha, 1 - sourceAlpha
	case Clear:
		return 0, 0
	default:
		return 1, 0
	}
}

func pointInRect(xPos, yPos float64, rect Rect) bool {
	return xPos >= rect.X && yPos >= rect.Y && xPos < rect.X+rect.Width && yPos < rect.Y+rect.Height
}

// BlitOpacity composites source onto the surface with a global opacity.
func (surface *Surface) BlitOpacity(offsetX, offsetY int, source *Surface, opacity float64) {
	if source == nil || opacity <= 0 {
		return
	}
	if opacity > 1 {
		opacity = 1
	}
	for sourceY := 0; sourceY < source.Height; sourceY++ {
		destinationY := offsetY + sourceY
		if destinationY < 0 || destinationY >= surface.Height {
			continue
		}
		for sourceX := 0; sourceX < source.Width; sourceX++ {
			destinationX := offsetX + sourceX
			if destinationX < 0 || destinationX >= surface.Width {
				continue
			}
			if !pointInRect(float64(destinationX)+0.5, float64(destinationY)+0.5, surface.clip) {
				continue
			}
			sourceOffset := sourceY*source.Stride + sourceX*4
			destinationOffset := destinationY*surface.Stride + destinationX*4
			sourceAlpha := float64(source.Pix[sourceOffset+3]) / 255 * opacity
			if sourceAlpha <= 0 {
				continue
			}
			destinationAlpha := float64(surface.Pix[destinationOffset+3]) / 255
			surface.Pix[destinationOffset] = toByte(float64(source.Pix[sourceOffset])/255*opacity + float64(surface.Pix[destinationOffset])/255*(1-sourceAlpha))
			surface.Pix[destinationOffset+1] = toByte(float64(source.Pix[sourceOffset+1])/255*opacity + float64(surface.Pix[destinationOffset+1])/255*(1-sourceAlpha))
			surface.Pix[destinationOffset+2] = toByte(float64(source.Pix[sourceOffset+2])/255*opacity + float64(surface.Pix[destinationOffset+2])/255*(1-sourceAlpha))
			surface.Pix[destinationOffset+3] = toByte(sourceAlpha + destinationAlpha*(1-sourceAlpha))
		}
	}
}
