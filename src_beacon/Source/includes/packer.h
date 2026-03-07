/* Maverick — Binary packer (BE) and parser (LE) API */

#ifndef MV_PACKER_H
#define MV_PACKER_H

typedef struct {
    unsigned char *data;
    int len;
    int cap;
} PackBuf;

void pb_init(PackBuf *pb);
void pb_byte(PackBuf *pb, unsigned char val);
void pb_int16be(PackBuf *pb, unsigned short val);
void pb_int32be(PackBuf *pb, unsigned int val);
void pb_pad(PackBuf *pb, unsigned char *data, int size);
void pb_bytes(PackBuf *pb, unsigned char *data, int size);
void pb_str(PackBuf *pb, char *str);

typedef struct {
    unsigned char *buf;
    int pos;
    int len;
} Parser;

unsigned char   parser_byte(Parser *p);
unsigned int    parser_int32le(Parser *p);
unsigned char  *parser_str(Parser *p, int *out_len);
unsigned char  *parser_raw(Parser *p, int size);

#endif
