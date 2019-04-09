
// tupperware base object class.

#ifndef _TW_OBJECT
#define _TW_OBJECT

namespace TWlib {

class tw_object {
	// need some RTTI capability
 public:

	virtual ~tw_object() = 0; // this is for garbage collection purposes
 protected:


};

}
#endif

