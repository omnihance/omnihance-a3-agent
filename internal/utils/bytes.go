package utils

func MakeFixedLengthStringBytes(str string, length int) []byte {
	bytesMsg := make([]byte, length)
	strBytes := []byte(str)
	copy(bytesMsg, strBytes)
	return bytesMsg
}
