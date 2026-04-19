// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package npwssdk

import (
	"crypto/rand"
	"fmt"
)

const defaultPaddingBlockSize = 128

// MtoPad applies custom block alignment padding used by NPWS.
// The last byte of each block stores the number of random padding bytes.
// Block size defaults to 128 bytes.
func MtoPad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 1 {
		blockSize = defaultPaddingBlockSize
	}

	// Calculate how many random bytes we need.
	// We need at least 1 byte for the length indicator.
	usable := blockSize - 1 // bytes available for data per block
	totalBlocks := (len(data) + usable) / usable
	if len(data) == 0 {
		totalBlocks = 1
	}
	// If data exactly fills blocks, no extra block needed — length byte fits in last position.

	totalSize := totalBlocks * blockSize
	randomLen := totalSize - len(data) - 1 // -1 for the length byte

	result := make([]byte, totalSize)
	copy(result, data)

	// Fill random bytes between data and length indicator
	if randomLen > 0 {
		randomBytes := make([]byte, randomLen)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("padding: generating random bytes: %w", err)
		}
		copy(result[len(data):], randomBytes)
	}

	// Last byte stores the number of random padding bytes
	result[totalSize-1] = byte(randomLen) //nolint:gosec // randomLen is guaranteed to be in range [0,blockSize]
	return result, nil
}

// MtoUnpad removes the custom block alignment padding.
func MtoUnpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("padding: empty data")
	}

	randomLen := int(data[len(data)-1])
	dataLen := len(data) - randomLen - 1

	if dataLen < 0 || dataLen > len(data) {
		return nil, fmt.Errorf("padding: invalid padding length %d for data of size %d", randomLen, len(data))
	}

	return data[:dataLen], nil
}
