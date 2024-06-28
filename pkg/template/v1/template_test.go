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

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pkcs12ContentNoPass   = `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`
	pkcs12ContentWithPass = `MIIJYQIBAzCCCScGCSqGSIb3DQEHAaCCCRgEggkUMIIJEDCCA8cGCSqGSIb3DQEHBqCCA7gwggO0AgEAMIIDrQYJKoZIhvcNAQcBMBwGCiqGSIb3DQEMAQYwDgQI2eZRJ7Ar+JQCAggAgIIDgFTbOtkFPjqxAoYRHoq1SbyXKf/NRbBA5AqxQlv9aFVT4VcxUSrMGaSWifX2UjsVWQzn134yoLLbdJ0jTorVD+EuBUmtb3xXbBwLqtFZxwcWodYA5WhPQdDcQo0cD3o1vrsXPQARQR6ISSFnhFjPYdH9cO2LqUKV5pjFhIs2/1VPDS2eY7SWZN52DK3QknSj23S3ZW2s4TFEj/5C4ssbO7cWNWBjjaORnd17FMNgVtcRw8ITmLdGBOpFUwP8wIdiLGrXiyjfMLns74nztRelV30/v0DPlz0pZtOPygi/dy0qpbil3wtOFrtQBLEdvLNmt9ikQgGs3pJBS68eMJLu3jAU6rCIKycq0+E0eMXeHcseyMwgguTj2h4t+E4S7nU11lViBFqkSBKxE28+9fNlPvCsZ4WhQZ6TAW3E/jDy/ZSqmak5V7/khMlRPvtrxz71ivksH0iipPdJJkGi7SDEvETySBETiqIslUmsF0ZYeHR5wIBkB5V8zmi8RRZtpvDGbzuQ22V6sNk2mTDh+BRus7gNCoSGWYXWqNNp1PnznuYCJp9T+0mObcAijE7IQuhpYMeQPF+MUIlG5lmpNouzuygTf++xrKIjzP36DcthnMPeD/8LYWfzkuAeRodjl7Z1G6XLvBD5h+Xlq23WPjMcXUiiWYXxTREAQ1EWUf4A9twGcxHJ5AatbvQY3QUoS4a7LNuy17lF7G+O1SFDtGeHZXHHecaVpuAtqZEYeUpqy6ZzMJXtXE1JNl/UR9TtTordb1V5Pf45JTYKLI+SwxVQbRiTgfhulNc+E3tV1AEELZt4CKmh1OFJoJRtyREMfdVuP4rx7ywIoMYuWw8CRqJ3qSmdwz2kmL2pfJNn6vDfN6rNa+sXWErOJ7naSAKQH2CJfnkCOFxkKfwjbOcNRbnGKah8NyWu6xqlv7Y4xEJYqmPahGHmrSfdAt3mpc5zD74DvetLIczcadKaLH7mp6h98xHayvXh0UE7ERHeskfSPrLxL9A3V1RZXDOtcZFWSKHzokuXMDF9OnrcMYDzYgtzof4ReY2t1ldGF7phYINhDlUNyhzyjwyWQbdkxr/+FtWq8Sbm7o2zMTby48c25gnqD9U8RTAO+bY3oV3dQ4bpAOzWdzVmFjESUFx0xVwbTSJGXdkH4YmD5He+xwxTa0Je0HE5+ui5dbP1gxUY+pCGLOhGMIIFQQYJKoZIhvcNAQcBoIIFMgSCBS4wggUqMIIFJgYLKoZIhvcNAQwKAQKgggTuMIIE6jAcBgoqhkiG9w0BDAEDMA4ECGYAccNFWW6kAgIIAASCBMgbXw69iyx73RGWS3FYeuvF5L1VWXQGrDLdUGwa3SdovXxmP1pKBAb82UuiydXDpKSVCmefeEQTs6ucEFRmXrDIldNanqUwbydy3GSJ+4iNsVQHbFgNppH4e90L7BlLcZ3MzSrVEwxWVMHq+XLqTKf3X3QyrmA3mmF96+Nd+icpDIktd+/h2BHnSXHNP0QfVH27H4DwbMDijttXY0JB+8qP9m25Wn4zkmOPEUhrY4Ptv2I08eHFAuNI0jWUwfRhC4FDbUdwFb0aZjA3Te6uYTsu2zAlmg9HuqsD/Ef/wkBEKZLBkjiXa/niFVrwELXhWZDPBAuo+/1UbzXglsW4QDU4LbUutcs6DLag1vLe40a2LO1ODQm7Zw0bxLkb3f/ry6ZFYvO78XmHo4c/oQf4KPUtM2bLz5q7uOxAx07vHYaU2BVt3NjgiIO5VVKjw0075GdgFxwPvYncv1fsC5jSIkX43GuzEtoBTpJKDYb2nhKbN9XWixwGOhUBTK3WYBhn+uaMJs4l3EgkDtK9tsUs5VQQHawj0WrGS1mQhaBfcyZzv4wSn0d3JUO2CN0e9EReJcQvsEnwUvohilOvjDHHhTq8Kp4XU4jbq7TAKqxs3TOmdoskRykn9oKUPExJVhJQonFT3ietV5BHrnN/QoDCSeOR80ZxvWHrQDz3Hm1ygiHd8LYmN4IjiD8b28ZrCALifWxh0WmIYtLZrUjMZavPh+caWH9IG32fTxV9b1bgJD8vWqscj9jCjeMJvkKQo8PFg1kMAxt1u+bIyktTq42O9qxwGrdqEMeBzXxDJMMaRIH3m9LNZ/P5Nk4/hMURhCZJtRtNfOVTK+Q6kKgsdK2EHcuEnp/qBefZjve+xmitbF1W7C4+B7b2JNBacdIm1nE56DwglT/IUk65JrNFP3rf4c5ic76LCQrvyfLiKCGaqcihM9siLVFPYdrnr8TlGbCFnGbpBqMQA5MtZQaDUug50PJtdxlgfwWH4qliimgchCaZbSTcgN5YTguSe16uUSusHD+r6XdtI0939uDILXJjQMczhIKNw8w0Tn4Z3/g2KlB6cwbtaglnnO4a/USh0cPC1a581byNqeFoMi+mAhqfKkwdDuti4GX7OrhkUOkiRjEUXdcckpmmIsyamH/g1dq3CNFXFNIgRRrzIDo4Opr3Ip2VE/4BDQoo/+Rybzxh8bsHgCEujQf8urGxjGyd2ulHoXzHWhz7pPPuY5UN6dC9WZmOQDVous/1nhYThoLVVc61Rk6d83+Ac7iRg4bY5q/73J4HvPMmrTOOOqqn3wc9Pe5ibEy4tFaYnim4p1ZRm8YcwosZmuFPdsP6G5l5qt6uOyr2+qNpXIBkDpG7I6Ls10O7L3PQAX9zRGfcz6Ds0KtuDrLpaVvhuXpewsBwpo1lmhv9bAa4ppBuWznmKigX+vYojSxd/eCRAtMs+Lx6ppZsYNVhbdEIGKXSGwG98sSTZkoLHBMkUW7S8jpeSCHZWEFBUOPJQzAr5cW1w+RAs33cGUygZ5XEEx4DeW8MnO4lCuP+VDOwu3TAKhzAD+qCyXbLEzWiyL5fq3XL+YJtoAc8Mra9lK6jDqzq4u+PLNoYY+kWTBhCyRZ+PfzcXLry8pxuP5E6VtRgfYcxJTAjBgkqhkiG9w0BCRUxFgQUOEXV6IFYGpCSHi0MPHz4b3W0KOQwMTAhMAkGBSsOAwIaBQAEFBa+SV9FU2UObo+nYKdyt/kZVw6FBAgey4GonFtJ2gICCAA=`
	pkcs12Cert            = `-----BEGIN CERTIFICATE-----
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
	pkcs12Key = `-----BEGIN PRIVATE KEY-----
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
	jwkPubRSA     = `{"kid":"ex","kty":"RSA","key_ops":["sign","verify","wrapKey","unwrapKey","encrypt","decrypt"],"n":"p2VQo8qCfWAZmdWBVaYuYb-a-tWWm78K6Sr9poCvNcmv8rUPSLACxitQWR8gZaSH1DklVkqz-Ed8Cdlf8lkDg4Ex5tkB64jRdC1Uvn4CDpOH6cp-N2s8hTFLqy9_YaDmyQS7HiqthOi9oVjil1VMeWfaAbClGtFt6UnKD0Vb_DvLoWYQSqlhgBArFJi966b4E1pOq5Ad02K8pHBDThlIIx7unibLehhDU6q3DCwNH_OOLx6bgNtmvGYJDd1cywpkLQ3YzNCUPWnfMBJRP3iQP_WI21uP6cvo0DqBPBM4wvVzHbCT0vnIflwkbgEWkq1FprqAitZlop9KjLqzjp9vyQ","e":"AQAB"}`
	jwkPubRSAPKIX = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAp2VQo8qCfWAZmdWBVaYu
Yb+a+tWWm78K6Sr9poCvNcmv8rUPSLACxitQWR8gZaSH1DklVkqz+Ed8Cdlf8lkD
g4Ex5tkB64jRdC1Uvn4CDpOH6cp+N2s8hTFLqy9/YaDmyQS7HiqthOi9oVjil1VM
eWfaAbClGtFt6UnKD0Vb/DvLoWYQSqlhgBArFJi966b4E1pOq5Ad02K8pHBDThlI
Ix7unibLehhDU6q3DCwNH/OOLx6bgNtmvGYJDd1cywpkLQ3YzNCUPWnfMBJRP3iQ
P/WI21uP6cvo0DqBPBM4wvVzHbCT0vnIflwkbgEWkq1FprqAitZlop9KjLqzjp9v
yQIDAQAB
-----END PUBLIC KEY-----
`
	jwkPrivRSA      = `{"kty" : "RSA","kid" : "cc34c0a0-bd5a-4a3c-a50d-a2a7db7643df","use" : "sig","n"   : "pjdss8ZaDfEH6K6U7GeW2nxDqR4IP049fk1fK0lndimbMMVBdPv_hSpm8T8EtBDxrUdi1OHZfMhUixGaut-3nQ4GG9nM249oxhCtxqqNvEXrmQRGqczyLxuh-fKn9Fg--hS9UpazHpfVAFnB5aCfXoNhPuI8oByyFKMKaOVgHNqP5NBEqabiLftZD3W_lsFCPGuzr4Vp0YS7zS2hDYScC2oOMu4rGU1LcMZf39p3153Cq7bS2Xh6Y-vw5pwzFYZdjQxDn8x8BG3fJ6j8TGLXQsbKH1218_HcUJRvMwdpbUQG5nvA2GXVqLqdwp054Lzk9_B_f1lVrmOKuHjTNHq48w","e"   : "AQAB","d"   : "ksDmucdMJXkFGZxiomNHnroOZxe8AmDLDGO1vhs-POa5PZM7mtUPonxwjVmthmpbZzla-kg55OFfO7YcXhg-Hm2OWTKwm73_rLh3JavaHjvBqsVKuorX3V3RYkSro6HyYIzFJ1Ek7sLxbjDRcDOj4ievSX0oN9l-JZhaDYlPlci5uJsoqro_YrE0PRRWVhtGynd-_aWgQv1YzkfZuMD-hJtDi1Im2humOWxA4eZrFs9eG-whXcOvaSwO4sSGbS99ecQZHM2TcdXeAs1PvjVgQ_dKnZlGN3lTWoWfQP55Z7Tgt8Nf1q4ZAKd-NlMe-7iqCFfsnFwXjSiaOa2CRGZn-Q","p"   : "4A5nU4ahEww7B65yuzmGeCUUi8ikWzv1C81pSyUKvKzu8CX41hp9J6oRaLGesKImYiuVQK47FhZ--wwfpRwHvSxtNU9qXb8ewo-BvadyO1eVrIk4tNV543QlSe7pQAoJGkxCia5rfznAE3InKF4JvIlchyqs0RQ8wx7lULqwnn0","q"   : "ven83GM6SfrmO-TBHbjTk6JhP_3CMsIvmSdo4KrbQNvp4vHO3w1_0zJ3URkmkYGhz2tgPlfd7v1l2I6QkIh4Bumdj6FyFZEBpxjE4MpfdNVcNINvVj87cLyTRmIcaGxmfylY7QErP8GFA-k4UoH_eQmGKGK44TRzYj5hZYGWIC8","dp"  : "lmmU_AG5SGxBhJqb8wxfNXDPJjf__i92BgJT2Vp4pskBbr5PGoyV0HbfUQVMnw977RONEurkR6O6gxZUeCclGt4kQlGZ-m0_XSWx13v9t9DIbheAtgVJ2mQyVDvK4m7aRYlEceFh0PsX8vYDS5o1txgPwb3oXkPTtrmbAGMUBpE","dq"  : "mxRTU3QDyR2EnCv0Nl0TCF90oliJGAHR9HJmBe__EjuCBbwHfcT8OG3hWOv8vpzokQPRl5cQt3NckzX3fs6xlJN4Ai2Hh2zduKFVQ2p-AF2p6Yfahscjtq-GY9cB85NxLy2IXCC0PF--Sq9LOrTE9QV988SJy_yUrAjcZ5MmECk","qi"  : "ldHXIrEmMZVaNwGzDF9WG8sHj2mOZmQpw9yrjLK9hAsmsNr5LTyqWAqJIYZSwPTYWhY4nu2O0EY9G9uYiqewXfCKw_UngrJt8Xwfq1Zruz0YY869zPN4GiE9-9rzdZB33RBw8kIOquY3MK74FMwCihYx_LiU2YTHkaoJ3ncvtvg"}`
	jwkPrivRSAPKCS8 = `-----BEGIN PRIVATE KEY-----
MIIEwAIBADANBgkqhkiG9w0BAQEFAASCBKowggSmAgEAAoIBAQCmN2yzxloN8Qfo
rpTsZ5bafEOpHgg/Tj1+TV8rSWd2KZswxUF0+/+FKmbxPwS0EPGtR2LU4dl8yFSL
EZq637edDgYb2czbj2jGEK3Gqo28ReuZBEapzPIvG6H58qf0WD76FL1SlrMel9UA
WcHloJ9eg2E+4jygHLIUowpo5WAc2o/k0ESppuIt+1kPdb+WwUI8a7OvhWnRhLvN
LaENhJwLag4y7isZTUtwxl/f2nfXncKrttLZeHpj6/DmnDMVhl2NDEOfzHwEbd8n
qPxMYtdCxsofXbXz8dxQlG8zB2ltRAbme8DYZdWoup3CnTngvOT38H9/WVWuY4q4
eNM0erjzAgMBAAECggEBAJLA5rnHTCV5BRmcYqJjR566DmcXvAJgywxjtb4bPjzm
uT2TO5rVD6J8cI1ZrYZqW2c5WvpIOeThXzu2HF4YPh5tjlkysJu9/6y4dyWr2h47
warFSrqK191d0WJEq6Oh8mCMxSdRJO7C8W4w0XAzo+Inr0l9KDfZfiWYWg2JT5XI
ubibKKq6P2KxND0UVlYbRsp3fv2loEL9WM5H2bjA/oSbQ4tSJtobpjlsQOHmaxbP
XhvsIV3Dr2ksDuLEhm0vfXnEGRzNk3HV3gLNT741YEP3Sp2ZRjd5U1qFn0D+eWe0
4LfDX9auGQCnfjZTHvu4qghX7JxcF40omjmtgkRmZ/kCgYEA4A5nU4ahEww7B65y
uzmGeCUUi8ikWzv1C81pSyUKvKzu8CX41hp9J6oRaLGesKImYiuVQK47FhZ++wwf
pRwHvSxtNU9qXb8ewo+BvadyO1eVrIk4tNV543QlSe7pQAoJGkxCia5rfznAE3In
KF4JvIlchyqs0RQ8wx7lULqwnn0CgYEAven83GM6SfrmO+TBHbjTk6JhP/3CMsIv
mSdo4KrbQNvp4vHO3w1/0zJ3URkmkYGhz2tgPlfd7v1l2I6QkIh4Bumdj6FyFZEB
pxjE4MpfdNVcNINvVj87cLyTRmIcaGxmfylY7QErP8GFA+k4UoH/eQmGKGK44TRz
Yj5hZYGWIC8CgYEAlmmU/AG5SGxBhJqb8wxfNXDPJjf//i92BgJT2Vp4pskBbr5P
GoyV0HbfUQVMnw977RONEurkR6O6gxZUeCclGt4kQlGZ+m0/XSWx13v9t9DIbheA
tgVJ2mQyVDvK4m7aRYlEceFh0PsX8vYDS5o1txgPwb3oXkPTtrmbAGMUBpECgYEA
mxRTU3QDyR2EnCv0Nl0TCF90oliJGAHR9HJmBe//EjuCBbwHfcT8OG3hWOv8vpzo
kQPRl5cQt3NckzX3fs6xlJN4Ai2Hh2zduKFVQ2p+AF2p6Yfahscjtq+GY9cB85Nx
Ly2IXCC0PF++Sq9LOrTE9QV988SJy/yUrAjcZ5MmECkCgYEAldHXIrEmMZVaNwGz
DF9WG8sHj2mOZmQpw9yrjLK9hAsmsNr5LTyqWAqJIYZSwPTYWhY4nu2O0EY9G9uY
iqewXfCKw/UngrJt8Xwfq1Zruz0YY869zPN4GiE9+9rzdZB33RBw8kIOquY3MK74
FMwCihYx/LiU2YTHkaoJ3ncvtvg=
-----END PRIVATE KEY-----
`
	jwkPubEC     = `{"kid":"https://kv-test-mj.vault.azure.net/keys/ec-p-521/e3d0e9c179b54988860c69c6ae172c65","kty":"EC","key_ops":["sign","verify"],"crv":"P-521","x":"AedOAtb7H7Oz1C_cPKI_R4CN_eai5nteY6KFW07FOoaqgQfVCSkQDK22fCOiMT_28c8LZYJRsiIFz_IIbQUW7bXj","y":"AOnchHnmBphIWXvanmMAmcCDkaED6ycW8GsAl9fQ43BMVZTqcTkJYn6vGnhn7MObizmkNSmgZYTwG-vZkIg03HHs"}`
	jwkPubECPKIX = `-----BEGIN PUBLIC KEY-----
MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQB504C1vsfs7PUL9w8oj9HgI395qLm
e15jooVbTsU6hqqBB9UJKRAMrbZ8I6IxP/bxzwtlglGyIgXP8ghtBRbtteMA6dyE
eeYGmEhZe9qeYwCZwIORoQPrJxbwawCX19DjcExVlOpxOQlifq8aeGfsw5uLOaQ1
KaBlhPAb69mQiDTccew=
-----END PUBLIC KEY-----
`
	jwkPrivEC      = `{"kty": "EC","kid": "rie3pHe8u8gjSa0IaJfqk7_iEfHeYfDYx-Bqi7vQc0s","crv": "P-256","x": "fDjg3Nq4jPf8IOZ0277aPVal_8iXySnzLUJAZghUzZM","y": "d863PeyBOK_Q4duiSmWwgIRzi1RPlFZTR-vACMlPg-Q","d": "jJs5xsoHUetdMabtt8H2KyX5T92nGul1chFeMT5hlr0"}`
	jwkPrivECPKCS8 = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgjJs5xsoHUetdMabt
t8H2KyX5T92nGul1chFeMT5hlr2hRANCAAR8OODc2riM9/wg5nTbvto9VqX/yJfJ
KfMtQkBmCFTNk3fOtz3sgTiv0OHbokplsICEc4tUT5RWU0frwAjJT4Pk
-----END PRIVATE KEY-----
`
)

func TestExecute(t *testing.T) {
	tbl := []struct {
		name        string
		tpl         map[string][]byte
		data        map[string][]byte
		expetedData map[string][]byte
		expErr      string
	}{
		{
			name: "test empty",
			tpl:  nil,
			data: nil,
		},
		{
			name: "base64decode func",
			tpl: map[string][]byte{
				"foo": []byte("{{ .secret | base64decode | toString }}"),
			},
			data: map[string][]byte{
				"secret": []byte("MTIzNA=="),
			},
			expetedData: map[string][]byte{
				"foo": []byte("1234"),
			},
		},
		{
			name: "fromJSON func",
			tpl: map[string][]byte{
				"foo": []byte("{{ $var := .secret | fromJSON }}{{ $var.foo }}"),
			},
			data: map[string][]byte{
				"secret": []byte(`{"foo": "bar"}`),
			},
			expetedData: map[string][]byte{
				"foo": []byte("bar"),
			},
		},
		{
			name: "from & toJSON func",
			tpl: map[string][]byte{
				"foo": []byte("{{ $var := .secret | fromJSON }}{{ $var.foo | toJSON }}"),
			},
			data: map[string][]byte{
				"secret": []byte(`{"foo": {"baz":"bang"}}`),
			},
			expetedData: map[string][]byte{
				"foo": []byte(`{"baz":"bang"}`),
			},
		},
		{
			name: "multiline template",
			tpl: map[string][]byte{
				"cfg": []byte(`
		datasources:
		- name: Graphite
			type: graphite
			access: proxy
			url: http://localhost:8080
			password: "{{ .password | toString }}"
			user: "{{ .user | toString }}"`),
			},
			data: map[string][]byte{
				"user":     []byte(`foobert`),
				"password": []byte("harharhar"),
			},
			expetedData: map[string][]byte{
				"cfg": []byte(`
		datasources:
		- name: Graphite
			type: graphite
			access: proxy
			url: http://localhost:8080
			password: "harharhar"
			user: "foobert"`),
			},
		},
		{
			name: "base64 pipeline",
			tpl: map[string][]byte{
				"foo": []byte(`{{ "123412341234" | toBytes | base64encode | base64decode | toString }}`),
			},
			data: map[string][]byte{},
			expetedData: map[string][]byte{
				"foo": []byte("123412341234"),
			},
		},
		{
			name: "base64 pkcs12 extract",
			tpl: map[string][]byte{
				"key":  []byte(`{{ .secret | base64decode | pkcs12key | pemPrivateKey }}`),
				"cert": []byte(`{{ .secret | base64decode | pkcs12cert | pemCertificate }}`),
			},
			data: map[string][]byte{
				"secret": []byte(pkcs12ContentNoPass),
			},
			expetedData: map[string][]byte{
				"key":  []byte(pkcs12Key),
				"cert": []byte(pkcs12Cert),
			},
		},
		{
			name: "base64 pkcs12 extract with password",
			tpl: map[string][]byte{
				"key":  []byte(`{{ .secret | base64decode | pkcs12keyPass "123456" | pemPrivateKey }}`),
				"cert": []byte(`{{ .secret | base64decode | pkcs12certPass "123456" | pemCertificate }}`),
			},
			data: map[string][]byte{
				"secret": []byte(pkcs12ContentWithPass),
			},
			expetedData: map[string][]byte{
				"key":  []byte(pkcs12Key),
				"cert": []byte(pkcs12Cert),
			},
		},
		{
			name: "base64 decode error",
			tpl: map[string][]byte{
				"key": []byte(`{{ .example | base64decode }}`),
			},
			data: map[string][]byte{
				"example": []byte("iam_no_base64"),
			},
			expErr: "unable to decode base64",
		},
		{
			name: "pkcs12 key wrong password",
			tpl: map[string][]byte{
				"key": []byte(`{{ .secret | base64decode | pkcs12keyPass "wrong" | pemPrivateKey }}`),
			},
			data: map[string][]byte{
				"secret": []byte(pkcs12ContentWithPass),
			},
			expErr: "unable to decode pkcs12",
		},
		{
			name: "pkcs12 cert wrong password",
			tpl: map[string][]byte{
				"cert": []byte(`{{ .secret | base64decode | pkcs12certPass "wrong" | pemCertificate }}`),
			},
			data: map[string][]byte{
				"secret": []byte(pkcs12ContentWithPass),
			},
			expErr: "unable to decode pkcs12",
		},
		{
			name: "fromJSON error",
			tpl: map[string][]byte{
				"key": []byte(`{{ "{ # no json # }" | toBytes | fromJSON }}`),
			},
			data:   map[string][]byte{},
			expErr: "unable to unmarshal json",
		},
		{
			name: "template syntax error",
			tpl: map[string][]byte{
				"key": []byte(`{{ #xx }}`),
			},
			data:   map[string][]byte{},
			expErr: "unable to parse template",
		},
		{
			name: "unknown key error",
			tpl: map[string][]byte{
				"key": []byte(`{{ .unknown }}`),
			},
			data:   map[string][]byte{},
			expErr: "unable to execute template at key key",
		},
		{
			name: "jwk rsa pub pem",
			tpl: map[string][]byte{
				"fn": []byte(`{{ .secret | jwkPublicKeyPem }}`),
			},
			data: map[string][]byte{
				"secret": []byte(jwkPubRSA),
			},
			expetedData: map[string][]byte{
				"fn": []byte(jwkPubRSAPKIX),
			},
		},
		{
			name: "jwk rsa priv pem",
			tpl: map[string][]byte{
				"fn": []byte(`{{ .secret | jwkPrivateKeyPem }}`),
			},
			data: map[string][]byte{
				"secret": []byte(jwkPrivRSA),
			},
			expetedData: map[string][]byte{
				"fn": []byte(jwkPrivRSAPKCS8),
			},
		},
		{
			name: "jwk ecdsa pub pem",
			tpl: map[string][]byte{
				"fn": []byte(`{{ .secret | jwkPublicKeyPem }}`),
			},
			data: map[string][]byte{
				"secret": []byte(jwkPubEC),
			},
			expetedData: map[string][]byte{
				"fn": []byte(jwkPubECPKIX),
			},
		},
		{
			name: "jwk ecdsa priv pem",
			tpl: map[string][]byte{
				"fn": []byte(`{{ .secret | jwkPrivateKeyPem }}`),
			},
			data: map[string][]byte{
				"secret": []byte(jwkPrivEC),
			},
			expetedData: map[string][]byte{
				"fn": []byte(jwkPrivECPKCS8),
			},
		},
	}

	for i := range tbl {
		row := tbl[i]
		t.Run(row.name, func(t *testing.T) {
			sec := &corev1.Secret{
				Data:       make(map[string][]byte),
				ObjectMeta: v1.ObjectMeta{Labels: make(map[string]string), Annotations: make(map[string]string)},
			}
			err := Execute(row.tpl, row.data, "", "", sec)
			if !ErrorContains(err, row.expErr) {
				t.Errorf("unexpected error: %s, expected: %s", err, row.expErr)
			}
			if row.expetedData == nil {
				return
			}
			assert.EqualValues(t, row.expetedData, sec.Data)
		})
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
