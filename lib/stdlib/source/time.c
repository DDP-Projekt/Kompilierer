/*
	This file implements extern functions from 
	Duden/Zeit.ddp
*/
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include "ddptypes.h"
#include "ddpmemory.h"
#include "ddpwindows.h"

#if DDPOS_LINUX
#define _POSIX_C_SOURCE 200809L
#endif // DDPOS_LINUX

#include <time.h>

ddpint Zeit_Seit_Programmstart() {
	return clock();
}

void Zeit_Lokal(ddpstring* ret) {
	// get time
	time_t t = time(NULL);
 	struct tm tm = *localtime(&t);

	// make string
	ret->str = NULL;
	ret->cap = 0;

	// format string
	char buff[30];
	int size = sprintf(buff, "%02d:%02d:%02d %02d.%02d.%02d", tm.tm_hour, tm.tm_min, tm.tm_sec, tm.tm_mday, tm.tm_mon + 1, tm.tm_year + 1900);
	
	// move buffer to dstr
	size += 1; // make room for null terminator
	ret->str = ddp_reallocate(ret->str, ret->cap, ret->cap + size); // add the read size to the string buffer
	memcpy(ret->str + ret->cap, buff, size); // copy the read data into the string
	ret->cap += size;
	ret->str[ret->cap-1] = '\0'; // null terminator hinzufügen
}

// crossplatform sleep function
void Warte(ddpfloat seconds)
{
    #ifdef DDPOS_WINDOWS
        Sleep(seconds * 1000);
    #elif _POSIX_C_SOURCE >= 199309L
        struct timespec ts;
        ts.tv_sec = seconds;
        ts.tv_nsec = seconds * 1000000000;
        nanosleep(&ts, NULL);
    #else
        usleep(seconds * 1000);
    #endif // DDPOS_WINDOWS
}