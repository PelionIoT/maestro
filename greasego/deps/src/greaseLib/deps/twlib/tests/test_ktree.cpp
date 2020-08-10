/** Test ktree.h **/

#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <ctype.h>
#include <unistd.h>

#include <TW/tw_alloc.h>

#include "ktree.h"
#include "TW/tw_ktree.h"

//using namespace ::std;


typedef const char *str_t;

KBTREE_INIT(str, str_t, kb_str_cmp)




void rand_str(char *dest, size_t length) {
    char charset[] = "0123456789"
                     "abcdefghijklmnopqrstuvwxyz"
                     "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

    while (length-- > 0) {
        size_t index = (double) rand() / RAND_MAX * (sizeof charset - 1);
        *dest++ = charset[index];
    }
    *dest = '\0';
}

static int N = 0;


void walkStr(const char **s) {
	printf("%d: %s\n",N++,*s);
}

void walkStr2(char **s) {
	printf("%d: %s\n",N++,*s);
}



const int NUM_STRINGS=100;
const int STR_SIZE = 10;


struct STR_CMP {
	int operator()(char *a, char *b) {
		return strcmp(a, b);
	} 
};

int main(int argc, char **argv) {
    int c;
    printf("test-kree\n");
    printf("------------------\n");

    while ( -1 != (c = ::getopt(argc,argv,"c")) )
    {
    	// switch (c)
    	// {
    	// 	case 'c':
    	// 	printf("Client mode.\n");
    	// 	client = true;
    	// 	break;
    	// }
    }



    char *randStrings[NUM_STRINGS];

    for (char*& s : randStrings) {
    	s = (char *) malloc(STR_SIZE);
    	rand_str(s,STR_SIZE);
    }


    kbtree_t(str) *h;

    h = kb_init(str, KB_DEFAULT_SIZE);

    for (char*& s : randStrings) {
    	kb_put(str, h, s);
    }



    __kb_traverse(str_t, h, walkStr);


//     for (char*& s : randStrings) {
// //    	char *ret = kb_get(str, h, s);
//     	free(ret);
//     }

    kb_destroy(str_t, h);

    printf("test-kree Object\n");
    printf("------------------\n");

    TWlib::TW_KTree_32<char*,char*,int,struct STR_CMP,TWlib::Alloc_Std> tree2;

    for (char*& s : randStrings) {
//    	kb_put(str, h, s);
    	tree2.put(&s);
    }

    tree2.traverse(walkStr2);

    for (char*& s : randStrings) {
//    	kb_put(str, h, s);
    	char *ret = tree2.del(&s);
    	free(ret);
    }


 }
