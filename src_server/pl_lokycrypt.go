package main

import (
	"crypto/rc4"
)

type MvCrypt struct {
	Key []byte
}

func NewMvCrypt(key []byte) *MvCrypt {
	k := make([]byte, len(key))
	copy(k, key)
	return &MvCrypt{Key: k}
}

func (mc *MvCrypt) Encrypt(data []byte) []byte {
	cipher, err := rc4.NewCipher(mc.Key)
	if err != nil {
		return nil
	}
	out := make([]byte, len(data))
	cipher.XORKeyStream(out, data)
	return out
}

func (mc *MvCrypt) Decrypt(data []byte) []byte {
	return mc.Encrypt(data)
}
