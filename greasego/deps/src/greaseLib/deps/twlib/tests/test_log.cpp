/*
 * test_log.cpp
 *
 *  Created on: Nov 14, 2011
 *      Author: ed
 *
 *      Test Alloc and Log
 */

#include <TW/tw_log.h>

int main() {
	TW_ERROR("Error test\n",NULL);
	TW_DEBUG("Debug log test\n",NULL);
	TW_DEBUG("TEST: %s\n","1234");
	TW_DEBUG_L("Debug log DEBUG_L\n",NULL);
	TW_DEBUG_LT("Debug log DEBUG_LT\n",NULL);
	TW_DEBUG_LT("Debug log DEBUG_LT: %s - %d \n","hello",4321);
	printf("Test __FILE__,__LINE___: %s:%d\n", __FILE__, __LINE__);

	for (int x=0;x<100000;x++)
		TW_DEBUG("hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh"
				"kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk",NULL);

	return 0;
}

