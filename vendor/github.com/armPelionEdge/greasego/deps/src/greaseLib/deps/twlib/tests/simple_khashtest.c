/*
 * simple_khashtest.c
 *
 *  Created on: Mar 1, 2012
 *      Author: ed
 * (c) 2012, Framez LLC
 */


#include <TW/khash.h>

#include <stdio.h>

KHASH_MAP_INIT_INT(32, char); // a hash map with int as key, and char as value

int main() {
	int ret, is_missing;
	khiter_t k;
	khash_t(32) *h = kh_init(32); // initializes the hash table. (khash_t(32) is type kh_32_t)

	k = kh_put(32, h, 5, &ret); // this creates key (int) 5, table h, of type '32' (yes, this is bizarre api design) - this returns an iterator
	if (!ret) kh_del(32, h, k);
	kh_value(h, k) = 10;
	k = kh_get(32, h, 10);
	is_missing = (k == kh_end(h));
	k = kh_get(32, h, 5);
	kh_del(32, h, k);
	for (k = kh_begin(h); k != kh_end(h); ++k)
		if (kh_exist(h, k)) kh_value(h, k) = 1;
	kh_destroy(32, h);
	return 0;
}


