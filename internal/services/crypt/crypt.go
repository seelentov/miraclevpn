// Package crypt provides cryptographic utilities for the application.
package crypt

type CryptService interface {
	GenerateHash(password string) (string, error)
	ComparePasswordAndHash(password, encodedHash string) (bool, error)
}
