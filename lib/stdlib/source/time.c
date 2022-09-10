/*
	defines inbuilt ddp functions to work with time
*/
#include <time.h>
#include "ddptypes.h"
#include "memory.h"

ddpint Zeit_Seit_Programmstart() {
	return clock();
}

ddpstring* Zeit_Lokal() {
	// get time
	time_t t = time(NULL);
 	struct tm tm = *localtime(&t);

	// make string
	ddpstring* dstr = ALLOCATE(ddpstring, 1);
	dstr->str = NULL;
	dstr->cap = 0;

	// format string
	char buff[30];
	int size = sprintf(buff, "%02d:%02d:%02d %02d.%02d.%02d", tm.tm_hour, tm.tm_min, tm.tm_sec, tm.tm_mday, tm.tm_mon + 1, tm.tm_year + 1900);
	
	// move buffer to dstr
	size += 1; // make room for null terminator
	dstr->str = reallocate(dstr->str, dstr->cap, dstr->cap + size); // add the read size to the string buffer
	memcpy(dstr->str + dstr->cap, buff, size); // copy the read data into the string
	dstr->cap += size;
	dstr->str[dstr->cap-1] = '\0'; // null terminator hinzufügen

	return dstr;
}