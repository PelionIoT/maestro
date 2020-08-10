/*
 * local_strdup.c
 *
 *  Created on: May 22, 2015
 *      Author: ed
 * (c) 2015, WigWag Inc.

	A simple work around for strdup() when also using XSI compatible functions - strerror_r
 */
/*
    MIT License

    Copyright (c) 2019, Arm Limited and affiliates.

    SPDX-License-Identifier: MIT
    
    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.
*/


#include <string.h>
#include <stdlib.h>
#include <stdio.h>


#define MAX_STRDUP 4096

char *local_strdup_safe(const char *s) {
	char *ret = NULL;
	if(s) {
		int n = strlen(s);
		if(n > MAX_STRDUP) {
			fprintf(stderr,"Overflow on local_strdup_safe() %d\n",n);
			n = MAX_STRDUP;
		}
		ret = (char *) malloc(n + 1);
		memcpy(ret,s,n);
		*(ret+n) = '\0';
	}
	return ret;
}

char *local_strcat_safe(const char *one, const char *two) {
	char *ret = NULL;
	if(one && two) {
		int n1 = strlen(one);
		int n2 = strlen(two);
		int n = n1 + n2;
		if(n > MAX_STRDUP) {
			fprintf(stderr,"Overflow on local_strcat_safe() %d\n",n);
			return NULL;
		}
		ret = (char *) malloc(n + 1);
		memcpy(ret,one,n1);
		memcpy(ret+n1,two,n2);
		*(ret+n) = '\0';
	}
	return ret;
}

#define MEMCPY_STATIC_STRING_APPEND( in ) {  \
      char *_walk = in;                       \
      while(*_walk != '\0' && space > 0) {    \
    	*walk_out = *_walk; _walk++; walk_out++;    	\
    	space--; out_space++;               \
      }  }

/**
 * Fast mempcy & JSON escape a string at the same time
 */
int memcpy_and_json_escape(char *out, const char *in, int in_len, int *out_len) {
	char *walk_out = out;
	const char *walk_in = in;
	int in_remain = in_len;
	int space = *out_len;
	int out_space = 0;

//	uint32_t codepoint = 0; // utf8 code point

	enum STATE { LOOKING, UTF8_BYTE2, UTF8_BYTE3, UTF8_BYTE4, ANSI_ESCAPE, ESCAPE };
	enum STATE state = LOOKING;

	while(in_remain > 0 && space > 0) {
		if(state == ESCAPE) {
			if(*walk_in == '[') { // that a 'CSI'
				state = ANSI_ESCAPE;
				walk_in++; in_remain++;
				continue;
			} else
				state = LOOKING;
		} else
		if(state == ANSI_ESCAPE) {
		// deal with CSI codes. we want to ignore them
		// good list here: https://en.wikipedia.org/wiki/ANSI_escape_code#CSI_codes
			if((*walk_in >= '0' && *walk_in <= '9') ||
					*walk_in == ';'
					) {
				walk_in++; in_remain++;
				continue;
			} else {
				// eat the last character - usually an 'm'
				state = LOOKING;
				walk_in++; in_remain++;
				continue;
			}

		}

		if(state == LOOKING) {
			if(*walk_in == 0x1B) {
				state = ESCAPE;
				walk_in++; in_remain++;
			} else
			if((*walk_in & 0x80) == 0) {
				switch(*walk_in) {
					case '\n':
						MEMCPY_STATIC_STRING_APPEND("\\n");
						walk_in++; in_remain--;
						break;
					case '\t':
						MEMCPY_STATIC_STRING_APPEND("\\n");
						walk_in++; in_remain--;
						break;
					case '\\':
						MEMCPY_STATIC_STRING_APPEND("\\\\");
						walk_in++; in_remain--;
						break;
					case '\r':
						MEMCPY_STATIC_STRING_APPEND("\\r");
						walk_in++; in_remain--;
						break;
					case '\f':
						MEMCPY_STATIC_STRING_APPEND("\\f");
						walk_in++; in_remain--;
						break;
					case '"':
						MEMCPY_STATIC_STRING_APPEND("\\\"");
						walk_in++; in_remain--;
						break;
					case '/':
						MEMCPY_STATIC_STRING_APPEND("\\/");
						walk_in++; in_remain--;
						break;
					default:
						if(*walk_in >= ' ') {
							// pass it as long as its not some control character
							*walk_out = *walk_in;
							walk_in++; walk_out++; space--; out_space++;
							in_remain--;
						} else {
							walk_in++; in_remain--;
						}
				}
			} else {
				// it's multi-byte UTF8 - got a high bit
				// its unicode - pass it
				*walk_out = *walk_in;
				walk_in++; walk_out++; space--; out_space++;
				in_remain--;
				// if we want to encode as \u - we would do all this crap here. skipping
// JSON standard says UTF8 is ok to put in quotes - skip this for now
//				codepoint = 0;
//				MEMCPY_STATIC_STRING_APPEND("\\u");
//				if((*walk_in >> 5) == 6) {
//					state = UTF8_BYTE2;
//					codepoint =
//				}
//				state = UTF8_BYTE2;
//				continue;
			}
		}
//		} else if(state == UTF8_BYTE2) {
//
//		}

	}
	*out_len = out_space;
	if(space > 0) *walk_out='\0';
	if(in_remain > 0)
		return -1;
	else
		return 0;
}

//#include <assert.h>
//int main() {
//	static const char * CSI = "\33[";
//
//	char *test1 = "123456789\"abcdef ads\"\121212\nq2121221END";
//	char *test2 = "123456789\"abcdef ads\"\121212\nq2121221END慑逑슢¢\\uABCDkjsa\\u0019\n\fEND";
//	char *test3 = "123456789\"¢\\¢abcdef ads\"\121212\nq2121221END\\\\\\\\\\END";
//	char *test4 = "\33[31mRED REDR RED\33[0mnormal normal慑逑슢¢\ END";
//	char *test5 = "\33[31;33fMOVE CURSOR HERE\33[0 normal all good ok END";
//	int out_len = 256;
//	char buf[256];
//	//  memcpy_and_json_escape(char *out, const char *in, int in_len, int *out_len)
//	memcpy_and_json_escape(buf,test1,strlen(test1),&out_len);
//	printf("test1:>>%s<<\n",test1);
//	printf("test1:>>%s<<\n\n",buf);
//	printf("strlen(buf)=%d, out_len=%d\n",(int)strlen(buf),out_len);
//	assert(strlen(buf)==out_len);
//
//	out_len = 256;
//	memcpy_and_json_escape(buf,test2,strlen(test2),&out_len);
//	printf("test2:>>%s<<\n",test2);
//	printf("test2:>>%s<<\n\n",buf);
//	assert(strlen(buf)==out_len);
//
//	out_len = 256;
//	memcpy_and_json_escape(buf,test3,strlen(test3),&out_len);
//	printf("test3:>>%s<<\n",test3);
//	printf("test3:>>%s<<\n\n",buf);
//
//	out_len = 256;
//	memcpy_and_json_escape(buf,test4,strlen(test4),&out_len);
//	printf("test4:>>%s<<\n",test4);
//	printf("test4:>>%s<<\n\n",buf);
//
////	out_len = 256;
////	memcpy_and_json_escape(buf,test5,strlen(test5),&out_len);
////	printf("test5:>>%s<<\n",test5);
////	printf("test5:>>%s<<\n\n",buf);
//
//}
//  gcc local_strdup.c -o test-str

