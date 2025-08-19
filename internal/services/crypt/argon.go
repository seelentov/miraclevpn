package crypt

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
)

type Argon2idParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type ArgonService struct {
	params *Argon2idParams
	logger *zap.Logger
}

func NewArgonService(params *Argon2idParams, logger *zap.Logger) *ArgonService {
	return &ArgonService{params, logger}
}

func (s *ArgonService) GenerateHash(password string) (string, error) {
	s.logger.Debug("generating hash", zap.Int("password_length", len(password)))
	salt := make([]byte, s.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		s.logger.Error("failed to generate salt", zap.Error(err))
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		s.params.Iterations,
		s.params.Memory,
		s.params.Parallelism,
		s.params.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		s.params.Memory,
		s.params.Iterations,
		s.params.Parallelism,
		b64Salt,
		b64Hash,
	)

	s.logger.Info("hash generated",
		zap.Int("salt_length", int(s.params.SaltLength)),
		zap.Int("key_length", int(s.params.KeyLength)),
	)
	return encodedHash, nil

}

func (s *ArgonService) ComparePasswordAndHash(password, encodedHash string) (bool, error) {
	s.logger.Debug("comparing password and hash", zap.Int("password_length", len(password)))
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		s.logger.Error("failed to decode hash", zap.Error(err))
		return false, err
	}

	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		s.logger.Info("password match", zap.Int("key_length", int(params.KeyLength)))
		return true, nil
	}
	s.logger.Warn("password does not match")
	return false, nil

}

func decodeHash(encodedHash string) (*Argon2idParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("неверный формат хеша")
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, errors.New("неверный алгоритм хеширования")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, errors.New("несовместимая версия Argon2")
	}

	params := &Argon2idParams{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return nil, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, err
	}
	params.SaltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, err
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}
