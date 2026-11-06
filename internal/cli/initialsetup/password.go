// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package initialsetup

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"
)

// passwordAlphabet omits look-alike characters (0/O, 1/l/I).
const passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

const (
	passwordGroups    = 5
	passwordGroupSize = 4
)

// clientHashPrefix marks a password already hashed the way the web client does.
const clientHashPrefix = "$CH$"

// GeneratePassword returns 5 groups of 4 random characters joined by "-".
// The dashes are part of the password. A nil source uses crypto/rand.
func GeneratePassword(source io.Reader) (string, error) {
	if source == nil {
		source = rand.Reader
	}
	limit := big.NewInt(int64(len(passwordAlphabet)))
	groups := make([]string, 0, passwordGroups)
	for g := 0; g < passwordGroups; g++ {
		var group strings.Builder
		for i := 0; i < passwordGroupSize; i++ {
			n, err := rand.Int(source, limit)
			if err != nil {
				return "", fmt.Errorf("generate admin password: %w", err)
			}
			group.WriteByte(passwordAlphabet[n.Int64()])
		}
		groups = append(groups, group.String())
	}
	return strings.Join(groups, "-"), nil
}

// ClientHash builds the web-client transport prehash used by Login /
// CompleteInitialSetup: "$CH$" + lowercase hex SHA-256 of
// "<password>:<username>" (UTF-8). It must stay aligned with
// hashPasswordClient in modules/auth/web. The server strips $CH$ and bcrypts
// the hex; this SHA-256 step is not the stored password hash.
func ClientHash(password string, username string) string {
	// codeql[go/weak-sensitive-data-hashing]: transport prehash matching the
	// web $CH$ contract; server bcrypts the hex (see ClientHash godoc).
	sum := sha256.Sum256([]byte(password + ":" + username))
	return clientHashPrefix + hex.EncodeToString(sum[:])
}
