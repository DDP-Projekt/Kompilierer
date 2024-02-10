#ifndef DDP_DDPOS_H
#define DDP_DDPOS_H

#if defined(_WIN32)
#define DDPOS_WINDOWS 1
#elif defined(__linux__)
#define DDPOS_LINUX 1
#else
#error Operating System not supported
#endif

#endif // DDP_DDPOS_H
