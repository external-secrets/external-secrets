/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mfa

import (
	"encoding/base32"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	input := base32.StdEncoding.EncodeToString([]byte("12345678912345678912"))
	table := map[time.Time]string{
		time.Date(1970, 1, 1, 0, 0, 59, 0, time.UTC):     "016480",
		time.Date(2005, 3, 18, 1, 58, 29, 0, time.UTC):   "785218",
		time.Date(2005, 3, 18, 1, 58, 31, 0, time.UTC):   "081980",
		time.Date(2009, 2, 13, 23, 31, 30, 0, time.UTC):  "369925",
		time.Date(2016, 9, 16, 12, 40, 12, 0, time.UTC):  "437634",
		time.Date(2033, 5, 18, 3, 33, 20, 0, time.UTC):   "413665",
		time.Date(2603, 10, 11, 11, 33, 20, 0, time.UTC): "151178",
	}

	for when, expected := range table {
		code, _, err := generateCode(WithToken(input), WithWhen(when))

		require.NoError(t, err)
		require.Equal(t, expected, code, when.String())
	}
}

func TestDifferentLength(t *testing.T) {
	input := base32.StdEncoding.EncodeToString([]byte("12345678912345678912"))
	table := map[time.Time]string{
		time.Date(1970, 1, 1, 0, 0, 59, 0, time.UTC):     "71016480",
		time.Date(2005, 3, 18, 1, 58, 29, 0, time.UTC):   "24785218",
		time.Date(2005, 3, 18, 1, 58, 31, 0, time.UTC):   "89081980",
		time.Date(2009, 2, 13, 23, 31, 30, 0, time.UTC):  "20369925",
		time.Date(2016, 9, 16, 12, 40, 12, 0, time.UTC):  "92437634",
		time.Date(2033, 5, 18, 3, 33, 20, 0, time.UTC):   "94413665",
		time.Date(2603, 10, 11, 11, 33, 20, 0, time.UTC): "91151178",
	}

	for when, expected := range table {
		code, _, err := generateCode(WithToken(input), WithWhen(when), WithLength(8))

		require.NoError(t, err)
		require.Equal(t, expected, code, when.String())
	}
}

func TestSpaceSeparatedToken(t *testing.T) {
	input := "asdf qwer zxcv fghj rtyu ghjk lk"
	table := map[time.Time]string{
		time.Date(1970, 1, 1, 0, 0, 59, 0, time.UTC):     "338356",
		time.Date(2005, 3, 18, 1, 58, 29, 0, time.UTC):   "474671",
		time.Date(2005, 3, 18, 1, 58, 31, 0, time.UTC):   "985005",
		time.Date(2009, 2, 13, 23, 31, 30, 0, time.UTC):  "453314",
		time.Date(2016, 9, 16, 12, 40, 12, 0, time.UTC):  "492092",
		time.Date(2033, 5, 18, 3, 33, 20, 0, time.UTC):   "797055",
		time.Date(2603, 10, 11, 11, 33, 20, 0, time.UTC): "385618",
	}

	for when, expected := range table {
		code, _, err := generateCode(WithToken(input), WithWhen(when))

		require.NoError(t, err)
		require.Equal(t, expected, code, when.String())
	}
}

func TestNonPaddedHashes(t *testing.T) {
	input := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	table := map[time.Time]string{
		time.Date(1970, 1, 1, 0, 0, 59, 0, time.UTC):     "983918",
		time.Date(2005, 3, 18, 1, 58, 29, 0, time.UTC):   "349978",
		time.Date(2005, 3, 18, 1, 58, 31, 0, time.UTC):   "074850",
		time.Date(2009, 2, 13, 23, 31, 30, 0, time.UTC):  "181361",
		time.Date(2016, 9, 16, 12, 40, 12, 0, time.UTC):  "296434",
		time.Date(2033, 5, 18, 3, 33, 20, 0, time.UTC):   "845675",
		time.Date(2603, 10, 11, 11, 33, 20, 0, time.UTC): "055244",
	}

	for when, expected := range table {
		code, _, err := generateCode(WithToken(input), WithWhen(when))

		require.NoError(t, err)
		require.Equal(t, expected, code, when.String())
	}
}

func TestInvalidPadding(t *testing.T) {
	input := "a6mr*&^&*%*&ylj|'[lbufszudtjdt42nh5by"
	table := map[time.Time]string{
		time.Date(1970, 1, 1, 0, 0, 59, 0, time.UTC):   "",
		time.Date(2005, 3, 18, 1, 58, 29, 0, time.UTC): "",
	}

	for when, expected := range table {
		code, _, err := generateCode(WithToken(input), WithWhen(when))

		require.Error(t, err)
		require.Equal(t, expected, code, when.String())
	}
}
