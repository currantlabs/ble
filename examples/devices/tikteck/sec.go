package main

import (
	"crypto/aes"
	"crypto/cipher"
	"log"
)

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

func newECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (x *ecbEncrypter) BlockSize() int { return x.blockSize }

func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

func pad(data []byte) []byte {
	padded := make([]byte, 16)
	copy(padded, data)
	return padded
}

func reverse(data []byte) []byte {
	sz := len(data)
	r := make([]byte, sz)
	for i, b := range data {
		r[sz-1-i] = b
	}
	return r
}

func xor(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

func genKey(name, pass []byte) []byte {
	k := make([]byte, 16)
	xor(k, pad(name), pad(pass))
	return k
}

func encrypt(key, data []byte) []byte {
	b, err := aes.NewCipher(reverse(key))
	if err != nil {
		log.Fatalf("can't create aes block: %s", err)
	}
	enc := make([]byte, 16)
	newECBEncrypter(b).CryptBlocks(enc, reverse(data))
	return reverse(enc)
}
