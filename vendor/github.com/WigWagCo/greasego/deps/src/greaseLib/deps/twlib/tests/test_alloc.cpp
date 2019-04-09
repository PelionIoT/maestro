/*
 * test_log.cpp
 *
 *  Created on: Nov 14, 2011
 *      Author: ed
 *
 *      Test Alloc and Log
 */

#include <TW/tw_log.h>
#include <TW/tw_alloc.h>

class TestClass {
public:
	TestClass() {
		TW_DEBUG("constructor.\n",NULL);
	}
	~TestClass() {
		TW_DEBUG("destructor.\n",NULL);
	}
	int x;
	char c[10];
};

int main() {
	TW_ERROR("Error test\n",NULL);
	TW_DEBUG("Debug log test\n",NULL);
	TW_DEBUG("TEST: %s\n","1234");
	TW_DEBUG_L("Debug log DEBUG_L\n",NULL);
	TW_DEBUG_LT("Debug log DEBUG_LT\n",NULL);
	TW_DEBUG_LT("Debug log DEBUG_LT: %s - %d \n","hello",4321);
	printf("Test __FILE__,__LINE___: %s:%d\n", __FILE__, __LINE__);

	TestClass *t;
	TW_NEW(t, TestClass, TestClass(), Allocator<Alloc_Std> );

	TW_DELETE(t, TestClass, Allocator<Alloc_Std>);

	Allocator<Alloc_Std> *alloc = new Allocator<Alloc_Std>();

	TW_NEW_WALLOC(t, TestClass, TestClass(), alloc );

	TW_DELETE(t, TestClass, Allocator<Alloc_Std>);

	delete alloc;

	Allocator<Alloc_Std> alloc2;
	Allocator<Alloc_Std> *a2 =  &alloc2;
	TW_NEW_WALLOC(t, TestClass, TestClass(), a2 );

	TW_DELETE(t, TestClass, Allocator<Alloc_Std>);

	TW_NEW_WALLOC(t, TestClass, TestClass(), (&alloc2) );

	TW_DELETE(t, TestClass, Allocator<Alloc_Std>);


	return 0;
}

