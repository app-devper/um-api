package utils

import (
	"context"
	"crypto/rand"
	"time"
)

func InitContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	return ctx, cancel
}

func GenerateCode(length int) string {
	const otpChars = "1234567890"
	buffer := make([]byte, length)
	_, err := rand.Read(buffer)
	if err != nil {
		return ""
	}
	otpCharsLength := len(otpChars)
	for i := 0; i < length; i++ {
		buffer[i] = otpChars[int(buffer[i])%otpCharsLength]
	}
	return string(buffer)
}

func GenerateRefId(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	buffer := make([]byte, length)
	_, err := rand.Read(buffer)
	if err != nil {
		return ""
	}
	otpCharsLength := len(alphabet)
	for i := 0; i < length; i++ {
		buffer[i] = alphabet[int(buffer[i])%otpCharsLength]
	}
	return string(buffer)
}
