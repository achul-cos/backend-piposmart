package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	algorithm  = "argon2id"
	version    = argon2.Version
	memory     = 64 * 1024
	timeCost   = 3
	threads    = 2
	keyLength  = 32
	saltLength = 16
)

func HashArgon2id(plain string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("buat salt password: %w", err)
	}
	return HashArgon2idWithSalt(plain, salt), nil
}

func HashArgon2idWithSalt(plain string, salt []byte) string {
	key := argon2.IDKey([]byte(plain), salt, timeCost, memory, threads, keyLength)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)
	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s", algorithm, version, memory, timeCost, threads, encodedSalt, encodedKey)
}

func VerifyArgon2id(plain, encoded string) (bool, error) {
	params, salt, key, err := decode(encoded)
	if err != nil {
		return false, err
	}

	comparisonKey := argon2.IDKey([]byte(plain), salt, params.timeCost, params.memory, params.threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(key, comparisonKey) == 1, nil
}

type decodedParams struct {
	memory   uint32
	timeCost uint32
	threads  uint8
}

func decode(encoded string) (decodedParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != algorithm {
		return decodedParams{}, nil, nil, errors.New("format hash password tidak didukung")
	}

	if parts[2] != fmt.Sprintf("v=%d", version) {
		return decodedParams{}, nil, nil, errors.New("versi Argon2id tidak didukung")
	}

	paramParts := strings.Split(parts[3], ",")
	if len(paramParts) != 3 {
		return decodedParams{}, nil, nil, errors.New("parameter Argon2id tidak valid")
	}

	parsed := map[string]uint64{}
	for _, part := range paramParts {
		key, value, found := strings.Cut(part, "=")
		if !found {
			return decodedParams{}, nil, nil, errors.New("parameter Argon2id tidak valid")
		}
		number, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return decodedParams{}, nil, nil, fmt.Errorf("parameter Argon2id %s tidak valid: %w", key, err)
		}
		parsed[key] = number
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return decodedParams{}, nil, nil, fmt.Errorf("decode salt password: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return decodedParams{}, nil, nil, fmt.Errorf("decode hash password: %w", err)
	}

	return decodedParams{
		memory:   uint32(parsed["m"]),
		timeCost: uint32(parsed["t"]),
		threads:  uint8(parsed["p"]),
	}, salt, key, nil
}
