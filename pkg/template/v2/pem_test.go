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

package template

import "testing"

const (
	certData = `-----BEGIN CERTIFICATE-----
MIIDHTCCAgWgAwIBAgIRAKC4yxy9QGocND+6avTf7BgwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAeFw0yMTAzMjAyMDA4MDhaFw0yMTAzMjAyMDM4
MDhaMBIxEDAOBgNVBAoTB0FjbWUgQ28wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQC3o6/JdZEqNbqNRkopHhJtJG5c4qS5d0tQ/kZYpfD/v/izAYum4Nzj
aG15owr92/11W0pxPUliRLti3y6iScTs+ofm2D7p4UXj/Fnho/2xoWSOoWAodgvW
Y8jh8A0LQALZiV/9QsrJdXZdS47DYZLsQ3z9yFC/CdXkg1l7AQ3fIVGKdrQBr9kE
1gEDqnKfRxXI8DEQKXr+CKPUwCAytegmy0SHp53zNAvY+kopHytzmJpXLoEhxq4e
ugHe52vXHdh/HJ9VjNp0xOH1waAgAGxHlltCW0PVd5AJ0SXROBS/a3V9sZCbCrJa
YOOonQSEswveSv6PcG9AHvpNPot2Xs6hAgMBAAGjbjBsMA4GA1UdDwEB/wQEAwIC
pDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQW
BBR00805mrpoonp95RmC3B6oLl+cGTAVBgNVHREEDjAMggpnb29ibGUuY29tMA0G
CSqGSIb3DQEBCwUAA4IBAQAipc1b6JrEDayPjpz5GM5krcI8dCWVd8re0a9bGjjN
ioWGlu/eTr5El0ffwCNZ2WLmL9rewfHf/bMvYz3ioFZJ2OTxfazqYXNggQz6cMfa
lbedDCdt5XLVX2TyerGvFram+9Uyvk3l0uM7rZnwAmdirG4Tv94QRaD3q4xTj/c0
mv+AggtK0aRFb9o47z/BypLdk5mhbf3Mmr88C8XBzEnfdYyf4JpTlZrYLBmDCu5d
9RLLsjXxhag8xqMtd1uLUM8XOTGzVWacw8iGY+CTtBKqyA+AE6/bDwZvEwVtsKtC
QJ85ioEpy00NioqcF0WyMZH80uMsPycfpnl5uF7RkW8u
-----END CERTIFICATE-----
`

	otherCert = `-----BEGIN CERTIFICATE-----
MIIBqjCCAU+gAwIBAgIRAPnGGsBUMbZhmh5QdnYdBmUwCgYIKoZIzj0EAwIwGjEY
MBYGA1UEAxMPaW50ZXJtZWRpYXRlLWNhMB4XDTIyMDIwOTEwMjUzMVoXDTIyMDIx
MDEwMjUzMVowDjEMMAoGA1UEAxMDZm9vMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcD
QgAEqnxdeInykx8JZsLi13rZLekoG2cosQ3F+2InVNy7hCQ7soMqdaJsGQ6LFtov
ogUFtOOTRWrunblqNWGZsowHbKOBgTB/MA4GA1UdDwEB/wQEAwIHgDAdBgNVHSUE
FjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwHQYDVR0OBBYEFLtundVbuKd73OWzo6SY
by0Ajeb2MB8GA1UdIwQYMBaAFCLg80J/bZBbOd+Y8+V94l5xM2zEMA4GA1UdEQQH
MAWCA2ZvbzAKBggqhkjOPQQDAgNJADBGAiEA4K4SbVNqrEtl7RfwBfJFMnWI+X8D
zMPMc4Xqzp2qTxcCIQDsySgtiakypZfWakpB49zJph0kLwGK8xhWvGMUw1N1/w==
-----END CERTIFICATE-----
`

	keyData = `-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC3o6/JdZEqNbqN
RkopHhJtJG5c4qS5d0tQ/kZYpfD/v/izAYum4NzjaG15owr92/11W0pxPUliRLti
3y6iScTs+ofm2D7p4UXj/Fnho/2xoWSOoWAodgvWY8jh8A0LQALZiV/9QsrJdXZd
S47DYZLsQ3z9yFC/CdXkg1l7AQ3fIVGKdrQBr9kE1gEDqnKfRxXI8DEQKXr+CKPU
wCAytegmy0SHp53zNAvY+kopHytzmJpXLoEhxq4eugHe52vXHdh/HJ9VjNp0xOH1
waAgAGxHlltCW0PVd5AJ0SXROBS/a3V9sZCbCrJaYOOonQSEswveSv6PcG9AHvpN
Pot2Xs6hAgMBAAECggEACTGPrmVNZDCWa1Y2hkJ0J7SoNcw+9O4M/jwMp4l/PD6P
I98S78LYLCZhPLK17SmjUcnFO1AXKW1JeFS2D/fjfP256guvcqQNjLFoioxcOhVb
ZGyd1Mi8JPqP5wfOj16gBeYDwTkjz9wqldcfiZaL9XoXetkZecbzR2JwC2FtIVuC
0njTjMNYpaBKnoLb8OTR0EQz7lYEo2MkQiWryz8wseONnFmdfh18p+p10YgCbuCH
qesrWfDLLxaxZelNtDhDngg9LoCLmarYy7BgShacmUEgJTZ/x3xFC75thK3ln0OY
+ktTgvVotYYaZi7qAjQiEsTvkTAPg5RMpQLd2UIWsQKBgQDCBp+1vURbwGzmTNUg
HMipD6WDFdLc9DCacx6+ZqsEPTMWQbCpVZrDKiY0Rjt5F+xOCyMr00J5RDJXRC0G
+L7NcJdywOFutT7vB+cmETg7l/6PHweNYBnE66706eTL/KVYZMi4tEinarPWhHmL
jasfdLANtpDjdWkRt299TkPRbQKBgQDyS8Rr7KZdv04Csqkf+ASmiJpT5R6Y72kc
3XYpKETyB2FyPZkuh/zInMut9SkkSI9O/jA3zf956jj6sF1DHvp7T8KkIp5OAQeD
J9AF65m2MnZfHFUeJ6ZQsggwMWqrD0ycIWP7YWtiBHH+D1wGkjYrssq+bvG/yNpA
LtqdKq9lhQKBgQCZA2hIhy61vRckuEsLvCdzTGeW7UsR/XGnHEqOlaEhArKbRsrv
gBdA+qiOaSTV5svw8E+YbE7sG6AnuhhYeyreEYEeeoZOLJmpIG5mUwYp2UBj1nC6
SaOI7OVZOGu7g09SWokBQQxbG4cgEfFY4Sym7fs5lVTGTP3Dfwppo6NQMQKBgQCo
J5NDP3Lafwk58BpV+H/pv8YzUUDh7M2rXbtCpxLqUdr8OOnVlEUISWFF8m5CIyVq
MhjuscWLK9Wtjba7/YTjDaDM3sW05xv6lyfU5ATCoNTr/zLHgcb4HAZ4w+L+otiN
RtMnxB2NYf5mzuwUF2cG/secUEzwyAlIH/xStSwTLQKBgQCRvqF+rqxnegoOgwVW
qrWPv06wXD8dW2FlPpY5GXqA0l6erSK3YsQQToRmbem9ibPD7bd5P4gNbWfxwK4C
Wt+1Rcb8OrDhDJbYz85bXBnPecKp4EN0b9SHO0/dsCqn2w30emc+9T/4m1ZDkpBd
BixHvI/EJ8YK3ta5WdJWKC6hnA==
-----END PRIVATE KEY-----
`
)

const (
	filterPrivateKey = "private key"
	filterCert       = "certificate"
)

func TestFilterPEM(t *testing.T) {
	type args struct {
		input   string
		pemType string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "extract cert / cert first",
			args: args{
				input:   certData + keyData,
				pemType: filterCert,
			},
			want: certData,
		},
		{
			name: "extract cert / key first",
			args: args{
				input:   keyData + certData,
				pemType: filterCert,
			},
			want: certData,
		},
		{
			name: "extract multiple certs",
			args: args{
				input:   keyData + certData + keyData + otherCert,
				pemType: filterCert,
			},
			want: certData + otherCert,
		},
		{
			name: "extract key",
			args: args{
				input:   keyData + certData,
				pemType: filterPrivateKey,
			},
			want: keyData,
		},
		{
			name: "key with junk",
			args: args{
				input:   certData + keyData + "some ---junk---",
				pemType: filterPrivateKey,
			},
			want: keyData,
		},
		{
			name: "begin/end with junk",
			args: args{
				// pem.Decode trims junk from the beginning of the input
				// so we are able to decode both cert & key
				input:   "some junk" + certData + keyData + "some ---junk---",
				pemType: filterPrivateKey,
			},
			want: keyData,
		},
		{
			name: "interleaved junk",
			args: args{
				// can parse cert but not key due to junk
				input:   certData + "some junk" + keyData,
				pemType: filterPrivateKey,
			},
			wantErr: true,
		},
		{
			name: "err when junk",
			args: args{
				input:   "---junk---",
				pemType: filterPrivateKey,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterPEM(tt.args.pemType, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterPEM() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("filterPEM() = %v, want %v", got, tt.want)
			}
		})
	}
}
