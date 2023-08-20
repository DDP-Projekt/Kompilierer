#ifndef DDP_RUNTIME_H
#define DDP_RUNTIME_H

void SignalHandler(int sig);

void ddp_init_runtime(int argc, char** argv);
void ddp_end_runtime();

#endif // DDP_RUNTIME_H