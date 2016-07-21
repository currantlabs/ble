package main

import (
	"crypto/aes"
	"crypto/cipher"
	"log"
)

type ecb struct {
	cipher.Block
}

func newECB(b cipher.Block) cipher.BlockMode {
	return &ecb{Block: b}
}

func (e *ecb) CryptBlocks(dst, src []byte) {
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		e.Encrypt(dst, src[:e.BlockSize()])
		src = src[e.BlockSize():]
		dst = dst[e.BlockSize():]
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
	newECB(b).CryptBlocks(enc, reverse(data))
	return reverse(enc)
}
