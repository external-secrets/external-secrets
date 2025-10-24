# prerequisite:
# install step cli
# from: https://github.com/smallstep/cli

all: ca disjunct-ca intermediate leaf  \
	pkcs12-nopass pkcs12-disjunct pkcs12-multibag pkcs12-withpass-1234

clean:
	rm *.{pfx,crt,key,pem}

ca:
	step certificate create root-ca \
		root-ca.crt root-ca.key \
		--profile root-ca --kty OKP --curve Ed25519 \
		--no-password --insecure -f

disjunct-ca:
	step certificate create disjunct-root-ca \
		disjunct-root-ca.crt disjunct-root-ca.key \
		--profile root-ca --kty OKP --curve Ed25519 \
		--no-password --insecure -f

intermediate:
	step certificate create intermediate-ca \
		intermediate-ca.crt intermediate-ca.key \
            --profile intermediate-ca \
			--ca ./root-ca.crt \
			--ca-key ./root-ca.key \
			--kty EC --curve P-256 \
			--no-password --insecure -f

leaf:
	step certificate create foo \
		foo.crt foo.key --profile leaf \
		--ca ./intermediate-ca.crt \
		--ca-key ./intermediate-ca.key \
		--no-password --insecure -f

pkcs12-nopass: ca intermediate leaf
	# deliberately in wrong order
	cat foo.crt root-ca.crt intermediate-ca.crt > chain.pem

	# create pkcs12
	openssl pkcs12 -export \
		-in chain.pem \
		-inkey foo.key \
		-out foo-nopass.pfx \
		-password pass:

pkcs12-disjunct: ca intermediate disjunct-ca leaf
	cat root-ca.crt intermediate-ca.crt disjunct-root-ca.crt > disjunct-chain.pem

	openssl pkcs12 -export \
		-in foo.crt \
		-certfile disjunct-chain.pem \
		-inkey foo.key \
		-out foo-disjunct-nopass.pfx \
		-password pass:

pkcs12-multibag: ca intermediate leaf
	# deliberately in wrong order, we're missing the leaf cert here
	cat root-ca.crt intermediate-ca.crt > intermediate-chain.pem

	openssl pkcs12 -export \
		-in foo.crt \
		-certfile intermediate-chain.pem \
		-inkey foo.key \
		-out foo-multibag-nopass.pfx \
		-password pass:

pkcs12-withpass-1234: ca intermediate leaf
	# deliberately in the wrong order
	cat foo.crt root-ca.crt intermediate-ca.crt > chain.pem

	# create pkcs12
	openssl pkcs12 -export \
		-in chain.pem \
		-inkey foo.key \
		-out foo-withpass-1234.pfx \
		-password pass:1234
