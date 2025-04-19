/*
	Wrapper to include Windows.h
*/
#ifndef DDP_DDPWINDOWS_H
#define DDP_DDPWINDOWS_H

#include "ddpos.h"

#ifdef DDPOS_WINDOWS

#define NOMINMAX
#define WIN32_LEAN_AND_MEAN
#define WINVER 0x0A00
#define _WIN32_WINNT 0x0A00

#include <Windows.h>

#endif // DDPOS_WINDOWS

#endif // DDP_DDPWINDOWS_H
