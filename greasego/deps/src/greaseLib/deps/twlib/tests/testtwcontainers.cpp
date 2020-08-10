// WigWag LLC
// (c) 2010
// testtwcontainers.cpp
// Author: ed

#include <iostream>

#include <TW/tw_alloc.h>
#include <TW/tw_fifo.h>
#include <TW/tw_stack.h>



namespace TWlibTESTS {

typedef Allocator<Alloc_Std> TESTAlloc;

class testdat2 {
public:
	testdat2(int _x = -1) : x( _x ) {  }
	~testdat2() { }
	testdat2( testdat2& d ) { x = d.x; }
	testdat2& operator=(const testdat2 &o) { this->x = o.x; return *this; }
	int x;
};

}

using namespace std;
using namespace TWlib;
using namespace TWlibTESTS;

int main() {

	int x, y, loop1 = 10;

	cout << "test tw_Stack<T> " << endl;

	TWlib::Stack<testdat2> teststack;

	for(x=0;x<loop1;x++) {
		testdat2 t(x+5);
		teststack.push(t);
	}

	testdat2 fillme;

	y= 5;

	while(teststack.pop(fillme)) {
		cout << "pop: " << fillme.x << endl;
		if((fillme.x % 2) > 0) {
			testdat2 t(100);
			teststack.push(t);
			y--;
		}
	}

	tw_safeFIFO<testdat2,TESTAlloc> testfifo;


	for(x=0;x<loop1;x++) {
		testdat2 t(x+5);
		testfifo.add(t);
	}

	tw_safeFIFO<testdat2,TESTAlloc>::iter testiter;
	testfifo.startIter(testiter);
	while(testiter.getNext(fillme)) {
		cout << "next: " << fillme.x << endl;
	}
	testfifo.releaseIter(testiter);

	cout << "assignment test" << endl;

	tw_safeFIFO<testdat2,TESTAlloc> copy;
	tw_safeFIFO<testdat2,TESTAlloc>::iter testitercopy;
	copy = testfifo;
	copy.startIter(testitercopy);
	while(testitercopy.getNext(fillme)) {
		cout << "next (copy): " << fillme.x << endl;
	}
	copy.releaseIter(testitercopy);

	copy.add(fillme);

	cout << "add one more - show copy again" << endl;

	copy.startIter(testitercopy);
	while(testitercopy.getNext(fillme)) {
		cout << "next (copy): " << fillme.x << endl;
	}
	copy.releaseIter(testitercopy);

	cout << "---original copy---" << endl;

	testfifo.startIter(testiter);
	while(testiter.getNext(fillme)) {
		cout << "next: " << fillme.x << endl;
	}
	testfifo.releaseIter(testiter);



}

