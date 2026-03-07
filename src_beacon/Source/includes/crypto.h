/* Maverick — RC4 encryption API */

#ifndef MV_CRYPTO_H
#define MV_CRYPTO_H

#include "config.h"

unsigned char *rc4_encrypt(unsigned char *data, int data_len, unsigned char *key, int *out_len);
unsigned char *rc4_decrypt(unsigned char *data, int data_len, unsigned char *key, int *out_len);

#endif
