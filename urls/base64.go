package urls

import "encoding/base64"

func decodeBase64Std(input string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(input)
}
