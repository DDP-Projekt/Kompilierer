#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/error.h"
#include "DDP/utf8/utf8.h"
#include <memory.h>

typedef struct {
	char *ptr;
	char *end_ptr; // points to the null terminator
	ddpstring *text;
	ddpint index;
} TextIterator;

void TextIterator_von_Text(TextIterator *ret, ddpstring *text) {
	ret->text = text;
	ret->ptr = DDP_GET_STRING_PTR(text);
	ret->end_ptr = &ret->ptr[text->cap - 1];
	if (text->cap <= 0) {
		ret->end_ptr = ret->ptr;
	}
	ret->index = 1;
}

ddpbool TextIterator_Zuende(TextIterator *it) {
	return it->ptr >= it->end_ptr;
}

ddpchar TextIterator_Buchstabe(TextIterator *it) {
	if (TextIterator_Zuende(it)) {
		return '\0';
	}

	uint32_t ret;
	utf8_string_to_char(it->ptr, &ret);
	return (ddpchar)ret;
}

ddpchar TextIterator_Naechster(TextIterator *it) {
	if (TextIterator_Zuende(it)) {
		return '\0';
	}

	uint32_t ret;
	it->ptr += utf8_string_to_char(it->ptr, &ret);
	it->index++;
	return (ddpchar)ret;
}

ddpint TextIterator_Verbleibend(TextIterator *it) {
	if (TextIterator_Zuende(it)) {
		return 0;
	}

	return (ddpint)utf8_strlen(it->ptr);
}

void TextIterator_Rest(ddpstring *ret, TextIterator *it) {
	if (TextIterator_Zuende(it)) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	ddp_string_from_constant(ret, it->ptr);
}

void TextIterator_Bisher(ddpstring *ret, TextIterator *it) {
	*ret = DDP_EMPTY_STRING;

	char *it_text_ptr = DDP_GET_STRING_PTR(it->text);
	if (it->ptr <= it_text_ptr) {
		return;
	}

	ret->cap = it->ptr - it_text_ptr + 1;
	if (!DDP_IS_SMALL_STRING(ret)) {
		ret->str = DDP_ALLOCATE(char, ret->cap);
		ret->refc = NULL;
	}
	char *ret_ptr = DDP_GET_STRING_PTR(ret);

	memcpy(ret_ptr, it_text_ptr, ret->cap - 1);
	ret_ptr[ret->cap - 1] = '\0';
}
