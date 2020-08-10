/*
 * autopointertest.cpp
 *
 *  Created on: Dec 31, 2009
 *      Author: ed
 */
#include <stdio.h>
#include <iostream>

#include <TW/tw_autopointer.h>

using namespace std;
using namespace TWlib;

namespace TWlibTest {
void callback1( void *d ) {
	printf("hit callback1: %x\n", d);
}

void callback2( void *d ) {
	printf("hit callback2: %x\n", d);
}

}

using namespace TWlibTest;

int main() {

	string *testData = new string("TEST DATA");
	autoPointer<string> Ap( testData );

	Ap.registerCallback(&callback1);
	Ap.registerCallback(&callback2);

	int x;
	for (x=0;x<10;x++) {
		//string *dat = (string *) Ap.getP();
		string *dat = Ap;
		autoPointer<string> &ref = Ap.dupP(); // dup the autopointer, requiring release below
		ref->insert( 0, 1, 'B');// test dup()ed deref
		dat->insert( 0, 1, 'A');
		Ap->insert(0,1, '+'); // non-reference count incrementing, dereference
		cout << "DATA: " << *dat << endl;
		ref.release(); // release the dup() call
	}
	printf("should call callbacks...\n");
	Ap.release(); // should detroy now
	printf("Callbacks done. Destructor working\n");


}
