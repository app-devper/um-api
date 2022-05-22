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

const otpChars = "1234567890"

func GenerateCode(length int) string {
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

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GenerateRefId(length int) string {
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
