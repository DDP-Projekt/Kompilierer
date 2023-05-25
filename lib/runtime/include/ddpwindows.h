/*
	Wrapper to include Windows.h
*/
#ifndef DDP_DDPWINDOWS_H
#define DDP_DDPWINDOWS_H

#include "ddpos.h"

#ifdef DDPOS_WINDOWS

#define NOMINMAX
#include <Windows.h>

#endif // DDPOS_WINDOWS

#endif // DDP_DDPWINDOWS_H