// stack.h
//: C04:tw_Stack.h
// Nested struct in linked list
#ifndef STACK_H
#define STACK_H

#ifndef NULL
#define NULL 0
#endif

/** Implements a stack - FILO - first in last out
 *
 * T must support:
 * assignment (operator=)
 * default constructor
 *
 */
namespace TWlib {

template<class T>
class Stack {
protected:
	struct Link {
		T data;
		Link* next;
		void initialize(T &dat, Link* nxt);
	}* head;
	void cleanup();
public:
	Stack();
	~Stack();
	void push(T &dat);
	bool peek(T &fill);
	bool pop(T &fill);
	int remaining();
protected:
	int cnt;
};


template <class T>
Stack<T>::Stack() :
	cnt(0),
	head(NULL) { }

template <class T>
Stack<T>::~Stack()  {
	cleanup();
}

template <class T>
void Stack<T>::Link::initialize(T &dat, Stack<T>::Link* nxt) {
  data = dat;
  next = nxt;
}


template <class T>
inline int Stack<T>::remaining() { return cnt; }


template <class T>
void Stack<T>::push(T &dat) {
	Link* newLink = new Link;
  newLink->initialize(dat, head);
  head = newLink;
  cnt++;
}

template <class T>
bool Stack<T>::peek( T &fill ) {
	if(head) {
		fill = head->data;
		return true;
	} else
		return false;
}

template <class T>
bool Stack<T>::pop(T &fill ) {
  if(head == 0) return false;
  cnt--;
  fill = head->data;
  Link* oldHead = head;
  head = head->next;
  delete oldHead;
  return true;
}

template<class T>
void Stack<T>::cleanup() {
	Link* cursor = head;
	while (head) {
		cursor = cursor->next;
		delete head;
		head = cursor;
	}
	head = 0; // Officially empty
	cnt = 0;
} ///:~

}


#endif // STACK_H ///:~



