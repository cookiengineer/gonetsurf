package colors

import "math"

// SRGB is an sRGB color with components in the unclamped 0-1 range. Values
// outside the range indicate an out-of-gamut color.
type SRGB struct {
	Red   float64
	Green float64
	Blue  float64
	Alpha float64
}

// ToSRGB converts the color to sRGB (Color 4 §10-§12). It returns false for
// color spaces that are parsed but not converted yet.
//
// Lab and LCH are interpreted in the D50 space required by Color 4; OKLab,
// OKLCH, display-p3, and XYZ are D65. The result is not gamut-mapped.
func (color Color) ToSRGB() (SRGB, bool) {
	if color.CurrentColor {
		return SRGB{}, false
	}
	alpha := clamp(color.Alpha, 0, 1)
	switch color.Space {
	case SpaceRGB:
		return SRGB{
			Red:   color.Components[0] / 255,
			Green: color.Components[1] / 255,
			Blue:  color.Components[2] / 255,
			Alpha: alpha,
		}, true
	case SpaceHSL:
		converted := color.hslToRGB()
		return converted.ToSRGB()
	case SpaceHWB:
		converted := color.hwbToRGB()
		return converted.ToSRGB()
	case SpaceLab:
		xyz := labToXYZ(color.Components[0], color.Components[1], color.Components[2])
		return xyzD50ToSRGB(xyz, alpha), true
	case SpaceLCH:
		chroma := color.Components[1]
		hue := color.Components[2] * math.Pi / 180
		xyz := labToXYZ(color.Components[0], chroma*math.Cos(hue), chroma*math.Sin(hue))
		return xyzD50ToSRGB(xyz, alpha), true
	case SpaceOKLab:
		linear := oklabToLinearSRGB(color.Components[0], color.Components[1], color.Components[2])
		return linearToSRGB(linear, alpha), true
	case SpaceOKLCH:
		chroma := color.Components[1]
		hue := color.Components[2] * math.Pi / 180
		linear := oklabToLinearSRGB(color.Components[0], chroma*math.Cos(hue), chroma*math.Sin(hue))
		return linearToSRGB(linear, alpha), true
	case SpaceCustom:
		return customToSRGB(color, alpha)
	default:
		return SRGB{}, false
	}
}

// RGBA8 converts the color to non-premultiplied 8-bit sRGB channels, clipping
// out-of-gamut components.
func (color Color) RGBA8() (red, green, blue, alpha uint8, ok bool) {
	converted, ok := color.ToSRGBGamutMapped()
	if !ok {
		return 0, 0, 0, 0, false
	}
	return channelByte(converted.Red), channelByte(converted.Green), channelByte(converted.Blue), channelByte(converted.Alpha), true
}

func customToSRGB(color Color, alpha float64) (SRGB, bool) {
	switch color.CustomSpace {
	case "srgb":
		return SRGB{
			Red:   color.Components[0],
			Green: color.Components[1],
			Blue:  color.Components[2],
			Alpha: alpha,
		}, true
	case "srgb-linear":
		return linearToSRGB([3]float64{color.Components[0], color.Components[1], color.Components[2]}, alpha), true
	case "display-p3":
		linearP3 := [3]float64{
			srgbToLinear(color.Components[0]),
			srgbToLinear(color.Components[1]),
			srgbToLinear(color.Components[2]),
		}
		xyz := multiplyMatrix(displayP3ToXYZ, linearP3)
		linear := multiplyMatrix(xyzToLinearSRGB, xyz)
		return linearToSRGB(linear, alpha), true
	case "a98-rgb":
		linearA98 := [3]float64{
			gammaToLinear(color.Components[0], 563.0/256.0),
			gammaToLinear(color.Components[1], 563.0/256.0),
			gammaToLinear(color.Components[2], 563.0/256.0),
		}
		xyz := multiplyMatrix(a98ToXYZ, linearA98)
		linear := multiplyMatrix(xyzToLinearSRGB, xyz)
		return linearToSRGB(linear, alpha), true
	case "prophoto-rgb":
		linearProPhoto := [3]float64{
			proPhotoToLinear(color.Components[0]),
			proPhotoToLinear(color.Components[1]),
			proPhotoToLinear(color.Components[2]),
		}
		xyzD50 := multiplyMatrix(proPhotoToXYZ, linearProPhoto)
		xyzD65 := multiplyMatrix(d50ToD65, xyzD50)
		linear := multiplyMatrix(xyzToLinearSRGB, xyzD65)
		return linearToSRGB(linear, alpha), true
	case "rec2020":
		linearRec2020 := [3]float64{
			rec2020ToLinear(color.Components[0]),
			rec2020ToLinear(color.Components[1]),
			rec2020ToLinear(color.Components[2]),
		}
		xyz := multiplyMatrix(rec2020ToXYZ, linearRec2020)
		linear := multiplyMatrix(xyzToLinearSRGB, xyz)
		return linearToSRGB(linear, alpha), true
	case "xyz", "xyz-d65":
		xyz := [3]float64{color.Components[0], color.Components[1], color.Components[2]}
		linear := multiplyMatrix(xyzToLinearSRGB, xyz)
		return linearToSRGB(linear, alpha), true
	case "xyz-d50":
		xyzD50 := [3]float64{color.Components[0], color.Components[1], color.Components[2]}
		xyzD65 := multiplyMatrix(d50ToD65, xyzD50)
		linear := multiplyMatrix(xyzToLinearSRGB, xyzD65)
		return linearToSRGB(linear, alpha), true
	default:
		return SRGB{}, false
	}
}

func xyzD50ToSRGB(xyz [3]float64, alpha float64) SRGB {
	xyzD65 := multiplyMatrix(d50ToD65, xyz)
	linear := multiplyMatrix(xyzToLinearSRGB, xyzD65)
	return linearToSRGB(linear, alpha)
}

func labToXYZ(lightness, firstAxis, secondAxis float64) [3]float64 {
	const (
		epsilon = 216.0 / 24389.0
		kappa   = 24389.0 / 27.0
	)
	fy := (lightness + 16) / 116
	fx := fy + firstAxis/500
	fz := fy - secondAxis/200
	inverse := func(value float64) float64 {
		cubed := value * value * value
		if cubed > epsilon {
			return cubed
		}
		return (116*value - 16) / kappa
	}
	return [3]float64{
		d50White[0] * inverse(fx),
		d50White[1] * inverse(fy),
		d50White[2] * inverse(fz),
	}
}

func oklabToLinearSRGB(lightness, firstAxis, secondAxis float64) [3]float64 {
	long := lightness + 0.3963377774*firstAxis + 0.2158037573*secondAxis
	medium := lightness - 0.1055613458*firstAxis - 0.0638541728*secondAxis
	short := lightness - 0.0894841775*firstAxis - 1.2914855480*secondAxis
	long = long * long * long
	medium = medium * medium * medium
	short = short * short * short
	return [3]float64{
		4.0767416621*long - 3.3077115913*medium + 0.2309699292*short,
		-1.2684380046*long + 2.6097574011*medium - 0.3413193965*short,
		-0.0041960863*long - 0.7034186147*medium + 1.7076147010*short,
	}
}

func gammaToLinear(value, gamma float64) float64 {
	if value < 0 {
		return -math.Pow(-value, gamma)
	}
	return math.Pow(value, gamma)
}

func proPhotoToLinear(value float64) float64 {
	const threshold = 16.0 / 512.0
	if math.Abs(value) <= threshold {
		return value / 16
	}
	if value < 0 {
		return -math.Pow(-value, 1.8)
	}
	return math.Pow(value, 1.8)
}

func rec2020ToLinear(value float64) float64 {
	const (
		alpha = 1.09929682680944
		beta  = 0.018053968510807
	)
	if value < beta*4.5 {
		return value / 4.5
	}
	return math.Pow((value+alpha-1)/alpha, 1/0.45)
}

func linearToSRGB(linear [3]float64, alpha float64) SRGB {
	return SRGB{
		Red:   linearToChannel(linear[0]),
		Green: linearToChannel(linear[1]),
		Blue:  linearToChannel(linear[2]),
		Alpha: alpha,
	}
}

func linearToChannel(value float64) float64 {
	if value <= 0.0031308 {
		return 12.92 * value
	}
	return 1.055*math.Pow(value, 1/2.4) - 0.055
}

func srgbToLinear(value float64) float64 {
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}

func multiplyMatrix(matrix [3][3]float64, vector [3]float64) [3]float64 {
	return [3]float64{
		matrix[0][0]*vector[0] + matrix[0][1]*vector[1] + matrix[0][2]*vector[2],
		matrix[1][0]*vector[0] + matrix[1][1]*vector[1] + matrix[1][2]*vector[2],
		matrix[2][0]*vector[0] + matrix[2][1]*vector[1] + matrix[2][2]*vector[2],
	}
}

func channelByte(value float64) uint8 {
	value = clamp(value, 0, 1)
	return uint8(math.Round(value * 255))
}

var (
	d50White = [3]float64{0.9642956764295677, 1, 0.8251046025104601}

	// Bradford-adapted D50 <-> D65 matrices (Color 4 §10).
	d50ToD65 = [3][3]float64{
		{0.955473421488075, -0.02309845494876471, 0.06325924320057072},
		{-0.0283697093338637, 1.0099953980813041, 0.021041441191917323},
		{0.012314014864481998, -0.020507649298898964, 1.330365926242124},
	}

	// Linear sRGB <-> XYZ D65 (Color 4 §10).
	xyzToLinearSRGB = [3][3]float64{
		{3.2409699419045226, -1.537383177570094, -0.4986107602930034},
		{-0.9692436362808796, 1.8759675015077202, 0.04155505740717559},
		{0.05563007969699366, -0.20397695888897652, 1.0569715142428786},
	}

	// Display P3 uses the D65 white point.
	displayP3ToXYZ = [3][3]float64{
		{0.4865709486482162, 0.26566769316909306, 0.1982172852343625},
		{0.2289745640697488, 0.6917385218365064, 0.079286914093745},
		{0, 0.04511338185890264, 1.043944368900976},
	}

	// Adobe RGB (1998) is a D65 space with a 563/256 gamma.
	a98ToXYZ = [3][3]float64{
		{0.5766690429101305, 0.1855582379065463, 0.1882286462349947},
		{0.29734497525053605, 0.6273635662554661, 0.07529145849399788},
		{0.02703136138641234, 0.07068885253582723, 0.9913375368376388},
	}

	// ProPhoto RGB is a D50 space with a 1.8 gamma and a linear toe.
	proPhotoToXYZ = [3][3]float64{
		{0.7977604896723027, 0.13518583717574031, 0.0313493495815248},
		{0.2880711282292934, 0.7118432178101014, 0.00008565396060525902},
		{0, 0, 0.8251046025104601},
	}

	// Rec. 2020 is a D65 space with the Rec. 2020 transfer function.
	rec2020ToXYZ = [3][3]float64{
		{0.6369580483012914, 0.14461690358620832, 0.1688809751641721},
		{0.2627002120112671, 0.6779980715188708, 0.05930171646986196},
		{0, 0.028072693049087428, 1.060985057710791},
	}
)

// ToSRGBGamutMapped converts color to sRGB and reduces OKLCH chroma until the
// result fits the sRGB gamut. The hue and lightness are retained; this is a
// simple chroma-clamp approximation of the Color 4 gamut-mapping algorithm.
func (color Color) ToSRGBGamutMapped() (SRGB, bool) {
	converted, ok := color.ToSRGB()
	if !ok {
		return SRGB{}, false
	}
	if inSRGBGamut(converted) {
		return converted, true
	}
	lightness, chroma, hue := srgbToOKLCH(converted)
	lightness = clamp(lightness, 0, 1)
	low := 0.0
	high := chroma
	best := linearToSRGB(oklabToLinearSRGB(lightness, 0, 0), converted.Alpha)
	for iteration := 0; iteration < 28; iteration++ {
		middle := (low + high) / 2
		candidate := oklchToSRGB(lightness, middle, hue, converted.Alpha)
		if inSRGBGamut(candidate) {
			low = middle
			best = candidate
		} else {
			high = middle
		}
	}
	return best, true
}

func inSRGBGamut(color SRGB) bool {
	const epsilon = 1e-4
	return color.Red >= -epsilon && color.Red <= 1+epsilon &&
		color.Green >= -epsilon && color.Green <= 1+epsilon &&
		color.Blue >= -epsilon && color.Blue <= 1+epsilon
}

func srgbToOKLCH(color SRGB) (lightness, chroma, hue float64) {
	red := srgbToLinear(color.Red)
	green := srgbToLinear(color.Green)
	blue := srgbToLinear(color.Blue)
	long := math.Cbrt(0.4122214708*red + 0.5363325363*green + 0.0514459929*blue)
	medium := math.Cbrt(0.2119034982*red + 0.6806995451*green + 0.1073969566*blue)
	short := math.Cbrt(0.0883024619*red + 0.2817188376*green + 0.6299787005*blue)
	lightness = 0.2104542553*long + 0.7936177850*medium - 0.0040720468*short
	firstAxis := 1.9779984951*long - 2.4285922050*medium + 0.4505937099*short
	secondAxis := 0.0259040371*long + 0.7827717662*medium - 0.8086757660*short
	chroma = math.Hypot(firstAxis, secondAxis)
	hue = math.Atan2(secondAxis, firstAxis)
	return lightness, chroma, hue
}

func oklchToSRGB(lightness, chroma, hue, alpha float64) SRGB {
	firstAxis := chroma * math.Cos(hue)
	secondAxis := chroma * math.Sin(hue)
	linear := oklabToLinearSRGB(lightness, firstAxis, secondAxis)
	return linearToSRGB(linear, alpha)
}
