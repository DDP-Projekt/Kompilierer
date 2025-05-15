#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/ddpwindows.h"
#include "DDP/error.h"
#include <assert.h>
#include <string.h>

#ifdef DDPOS_WINDOWS
#include <sys/unistd.h>
#include <winsock2.h>
#include <ws2tcpip.h>
#else
#include <arpa/inet.h>
#include <netdb.h>
#include <netinet/in.h>
#include <poll.h>
#include <sys/socket.h>
#include <sys/time.h>
#include <sys/types.h>
#include <unistd.h>
#endif // DDPOS_WINDOWS

static_assert(sizeof(ddpchar) == sizeof(int), "sizeof(ddpchar) == sizeof(int)");

static_assert(AF_UNSPEC == 0, "AF_UNSPEC == 0");
static_assert(AF_INET == 2, "AF_INET == 2");
#ifdef DDPOS_WINDOWS
static_assert(AF_INET6 == 23, "AF_INET6 == 23");
#else
static_assert(AF_INET6 == 10, "AF_INET6 == 10");
#endif // DDPOS_WINDOWS

static_assert(SOCK_STREAM == 1, "SOCK_STREAM == 1");
static_assert(SOCK_DGRAM == 2, "SOCK_DGRAM == 2");

extern ddpbool ddp_netzwerk_timout;
#define DDP_NETWORK_MIGHT_TIMEOUT \
	{ ddp_netzwerk_timout = false; }
#define DDP_NETWORK_DID_TIMEOUT \
	{ ddp_netzwerk_timout = true; }

typedef struct ddpsockaddr_storage_padding {
	ddpint padding1;
	ddpint padding2;
	ddpint padding3;
	ddpint padding4;
	ddpint padding5;
	ddpint padding6;
	ddpint padding7;
	ddpint padding8;
	ddpint padding9;
	ddpint padding10;
	ddpint padding11;
	ddpint padding12;
	ddpint padding13;
	ddpint padding14;
	ddpint padding15;
	ddpint padding16;
} ddpsockaddr_storage_padding;

typedef struct ddpsockaddr_storage {
	ddpsockaddr_storage_padding storage;
	socklen_t size;
} ddpsockaddr_storage;

static_assert(sizeof(socklen_t) == sizeof(int), "sizeof(socklen_t) == sizeof(int)");
static_assert(sizeof(struct sockaddr_storage) == sizeof(ddpsockaddr_storage_padding), "sizeof (struct sockaddr_storage) == sizeof(ddp_sockaddr_storage)");

typedef struct ddpsocket {
	int fd;
	int family;
	int type;
} ddpsocket;

typedef struct ddpsocketlist {
	ddpsocket *arr;
	ddpint len;
	ddpint cap;
} ddpsocketlist;

typedef ddpsocketlist *ddpsocketlistref;

#ifdef DDPOS_WINDOWS

static void cleanup_winsock(void) {
	WSACleanup();
}

ddpbool Init_Windows(void) {
	WSADATA wsaData;

	if (WSAStartup(MAKEWORD(2, 2), &wsaData) != 0) {
		ddp_error("WSAStartup failed", false);
		return false;
	}

	if (LOBYTE(wsaData.wVersion) != 2 ||
		HIBYTE(wsaData.wVersion) != 2) {
		ddp_error("Version 2.2 of Winsock not available", false);
		WSACleanup();
		return false;
	}

	atexit(cleanup_winsock);

	return true;
}

void Schließe_Socket_Windows(ddpsocket *sock) {
	closesocket(sock->fd);
}

#else

void Schließe_Socket_Windows(ddpsocket *sock) {}

ddpbool Init_Windows(void) {
	return true;
}

#endif // DDPOS_WINDOWS

struct addrinfo *Lade_AddressInfo(int family, int type, const ddpstring *name, const ddpstring *service) {
	DDP_MIGHT_ERROR;
	int status;
	struct addrinfo hints;
	struct addrinfo *servinfo;		 // will point to the results

	memset(&hints, 0, sizeof hints); // make sure the struct is empty
	hints.ai_family = family;		 // IPv4 or IPv6
	hints.ai_socktype = type;		 // TCP/UDP
	hints.ai_flags = AI_PASSIVE;	 // fill in the ip in case of null

	if ((status = getaddrinfo(name->str, service->str, &hints, &servinfo)) != 0) {
		ddp_error("getaddrinfo Fehler: %s", false, gai_strerror(status));
		return NULL;
	}

	return servinfo;
}

void AddressInfo_zu_Text(ddpstring *ret, const struct addrinfo *info) {
	DDP_MIGHT_ERROR;
	void *addr;
	char ipstr[INET6_ADDRSTRLEN];

	// get the pointer to the address itself,
	// different fields in IPv4 and IPv6:
	if (info->ai_family == AF_INET) { // IPv4
		struct sockaddr_in *ipv4 = (struct sockaddr_in *)info->ai_addr;
		addr = &(ipv4->sin_addr);
	} else { // IPv6
		struct sockaddr_in6 *ipv6 = (struct sockaddr_in6 *)info->ai_addr;
		addr = &(ipv6->sin6_addr);
	}

	// convert the IP to a string and print it:
	if (inet_ntop(info->ai_family, addr, ipstr, sizeof ipstr) == NULL) {
#ifdef DDPOS_WINDOWS
		ddp_error_win("inet_ntop Fehler: ");
#else
		ddp_error("inet_ntop Fehler: ", true);
#endif
		*ret = DDP_EMPTY_STRING;
		return;
	}

	ddp_string_from_constant(ret, ipstr);
}

void Socket_Erstellen(ddpsocket *ret, int family, int type) {
	DDP_MIGHT_ERROR;
	int sock = socket(family, type, 0);
	if (sock < 0) {
#ifdef DDPOS_WINDOWS
		ddp_error_win("socket Fehler: ");
#else
		ddp_error("socket Fehler: ", true);
#endif
	}
	ret->fd = sock;
	ret->family = family;
	ret->type = type;
}

void Socket_Binden(const ddpsocket *sock, struct addrinfo *info) {
	DDP_MIGHT_ERROR;
	for (struct addrinfo *p = info; p != NULL; p = p->ai_next) {
		if (bind(sock->fd, info->ai_addr, info->ai_addrlen) < 0) {
#ifdef DDPOS_WINDOWS
			ddp_error_win("bind Fehler: ");
#else
			ddp_error("bind Fehler: ", true);
#endif
		} else {
			break;
		}
	}
}

void Socket_Verbinden(const ddpsocket *sock, struct addrinfo *info) {
	DDP_MIGHT_ERROR;
	for (struct addrinfo *p = info; p != NULL; p = p->ai_next) {
		if (connect(sock->fd, info->ai_addr, info->ai_addrlen) < 0) {
#ifdef DDPOS_WINDOWS
			ddp_error_win("connect Fehler: ");
#else
			ddp_error("connect Fehler: ", true);
#endif
		} else {
			break;
		}
	}
}

void Socket_Zuhoeren(const ddpsocket *sock, ddpint backlog) {
	DDP_MIGHT_ERROR;
	if (listen(sock->fd, (int)backlog) < 0) {
#ifdef DDPOS_WINDOWS
		ddp_error_win("listen Fehler: ");
#else
		ddp_error("listen Fehler: ", true);
#endif
	}
}

void Socket_Verbindung_Annehmen(ddpsocket *ret, const ddpsocket *sock, ddpsockaddr_storage *client_addr) {
	DDP_MIGHT_ERROR;
	client_addr->size = sizeof client_addr->storage;
	int conn = accept(sock->fd, (struct sockaddr *)&client_addr->storage, &client_addr->size);
	if (conn < 0) {
#ifdef DDPOS_WINDOWS
		ddp_error_win("accept Fehler: ");
#else
		ddp_error("accept Fehler: ", true);
#endif
	}
	ret->fd = conn;
	ret->family = ((struct sockaddr_storage *)client_addr)->ss_family;
	ret->type = sock->type;
}

ddpint Socket_Senden(const ddpsocket *sock, const ddpbytelistref data) {
	DDP_MIGHT_ERROR;
	DDP_NETWORK_MIGHT_TIMEOUT;
	ddpint result = 0;
	while (result < data->len) {
		int sent;
		if ((sent = send(sock->fd, (const char *)&data->arr[result], data->len - result, 0)) < 0) {
#ifdef DDPOS_WINDOWS
			int err = WSAGetLastError();
			if (err == WSAEWOULDBLOCK || err == WSAETIMEDOUT) {
				DDP_NETWORK_DID_TIMEOUT;
				return sent;
			}

			ddp_error_win("send Fehler: ");
#else
			if (errno == EAGAIN || errno == EWOULDBLOCK) {
				DDP_NETWORK_DID_TIMEOUT;
				return sent;
			}

			ddp_error("send Fehler: ", true);
#endif
			return result;
		}
		result += sent;
	}
	return result;
}

void Socket_Empfangen(ddpbytelist *ret, ddpsocket *sock, const ddpint max) {
	DDP_MIGHT_ERROR;
	DDP_NETWORK_MIGHT_TIMEOUT;
	ddpbyte *buf = DDP_ALLOCATE(ddpbyte, max);
	ddpint got = (ddpint)recv(sock->fd, (char *)buf, max, 0);
	if (got < 0) {
		*ret = DDP_EMPTY_LIST(ddpbytelist);
		DDP_FREE_ARRAY(ddpbyte, buf, max);
#ifdef DDPOS_WINDOWS
		int err = WSAGetLastError();
		if (err == WSAEWOULDBLOCK || err == WSAETIMEDOUT) {
			DDP_NETWORK_DID_TIMEOUT;
			return;
		}

		ddp_error_win("recv Fehler: ");
#else
		if (errno == EAGAIN || errno == EWOULDBLOCK) {
			DDP_NETWORK_DID_TIMEOUT;
			return;
		}

		ddp_error("recv Fehler: ", true);
#endif
		return;
	}

	if (got == 0) {
		*ret = DDP_EMPTY_LIST(ddpbytelist);
		DDP_FREE_ARRAY(ddpbyte, buf, max);
		return;
	}

	ret->cap = got;
	ret->len = got;
	ret->arr = ddp_reallocate(buf, max, ret->len);
}

ddpint Socket_Senden_An_Klient(const ddpsocket *sock, const ddpbytelistref data, const ddpsockaddr_storage *client) {
	DDP_MIGHT_ERROR;
	DDP_NETWORK_MIGHT_TIMEOUT;
	ddpint sent = 0;
	if ((sent = (ddpint)sendto(sock->fd, (const char *)data->arr, data->len, 0, (struct sockaddr *)&client->storage, client->size)) < 0) {
#ifdef DDPOS_WINDOWS
		int err = WSAGetLastError();
		if (err == WSAEWOULDBLOCK || err == WSAETIMEDOUT) {
			DDP_NETWORK_DID_TIMEOUT;
			return sent;
		}

		ddp_error_win("sendto Fehler: ");
#else
		if (errno == EAGAIN || errno == EWOULDBLOCK) {
			DDP_NETWORK_DID_TIMEOUT;
			return sent;
		}

		ddp_error("sendto Fehler: ", true);
#endif
	}

	return sent;
}

ddpint Socket_Senden_An(const ddpsocket *sock, const ddpbytelistref data, const struct addrinfo *info) {
	DDP_MIGHT_ERROR;
	ddpsockaddr_storage client;
	client.size = info->ai_addrlen;
	memcpy(&client.storage, info->ai_addr, info->ai_addrlen);
	return Socket_Senden_An_Klient(sock, data, &client);
}

void Socket_Empfangen_Von(ddpbytelist *ret, ddpsocket *sock, ddpint max, ddpsockaddr_storage *client_addr) {
	DDP_MIGHT_ERROR;
	DDP_NETWORK_MIGHT_TIMEOUT;
	ddpbyte *buf = DDP_ALLOCATE(ddpbyte, max);
	client_addr->size = sizeof(client_addr->storage);
	ddpint got = (ddpint)recvfrom(sock->fd, (char *)buf, max, 0, (struct sockaddr *)&client_addr->storage, &client_addr->size);
	if (got < 0) {
		*ret = DDP_EMPTY_LIST(ddpbytelist);
		DDP_FREE_ARRAY(char, buf, max);
#ifdef DDPOS_WINDOWS
		int err = WSAGetLastError();
		if (err == WSAEWOULDBLOCK || err == WSAETIMEDOUT) {
			DDP_NETWORK_DID_TIMEOUT;
			return;
		}

		ddp_error_win("recvfrom Fehler: ");
#else
		if (errno == EAGAIN || errno == EWOULDBLOCK) {
			DDP_NETWORK_DID_TIMEOUT;
			return;
		}

		ddp_error("recvfrom Fehler: ", true);
#endif
		return;
	}
	ret->cap = got;
	ret->len = got;
	ret->arr = ddp_reallocate(buf, max, ret->len);
}

void Socket_Timeout_Setzen(ddpsocket *sock, ddpint timeout, ddpbool send) {
	int opt = SO_RCVTIMEO;
	if (send) {
		opt = SO_SNDTIMEO;
	}

#ifdef DDPOS_WINDOWS
	DWORD timeout_val = timeout;
	int optlen = sizeof(DWORD);
#else
	struct timeval timeout_val;
	timeout_val.tv_sec = timeout / 1000;
	timeout_val.tv_usec = (timeout - timeout_val.tv_sec) * 1000;
	int optlen = sizeof(struct timeval);
#endif // DDPOS_WINDOWS

	if (setsockopt(sock->fd, SOL_SOCKET, opt, (const char *)&timeout_val, optlen) < 0) {
#ifdef DDPOS_WINDOWS
		ddp_error_win("recvfrom Fehler: ");
#else
		ddp_error("recvfrom Fehler: ", true);
#endif
	}
}

#define POLL_ERR -1
#define POLL_TIMEOUT 0

typedef struct PollFd {
	ddpsocket sock;
	ddpbyte events;
} PollFd;

typedef struct PollFdList {
	PollFd *arr;
	ddpint len;
	ddpint cap;
} PollFdList;

typedef PollFdList *PollFdListref;

#define POLL_EVENT_IN 0x01	 // Data to read
#define POLL_EVENT_OUT 0x02	 // Ready to write
#define POLL_EVENT_ERR 0x04	 // Error condition
#define POLL_EVENT_HUP 0x08	 // Hung up (peer closed)
#define POLL_EVENT_NVAL 0x10 // Invalid fd

#ifdef DDPOS_WINDOWS
static ddpint poll_winsock(PollFd *fds, ddpint nfds, ddpint timeout) {
	struct pollfd *pfds = DDP_ALLOCATE(struct pollfd, nfds);

	for (int i = 0; i < nfds; i++) {
		pfds[i].fd = fds[i].sock.fd;
		pfds[i].events = 0;
		if (fds[i].events & POLL_EVENT_IN) {
			pfds[i].events |= POLLIN;
		}
		if (fds[i].events & POLL_EVENT_OUT) {
			pfds[i].events |= POLLOUT;
		}
		pfds[i].revents = 0;
	}

	int ret = WSAPoll(pfds, nfds, timeout);
	if (ret == SOCKET_ERROR) {
		DDP_FREE_ARRAY(struct pollfd, pfds, nfds);
		ddp_error_win("WSAPoll Fehler: ");
		return POLL_ERR;
	}
	if (ret == 0) {
		DDP_FREE_ARRAY(struct pollfd, pfds, nfds);
		return POLL_TIMEOUT;
	}

	for (int i = 0; i < nfds; i++) {
		fds[i].events = 0;
		if (pfds[i].revents & POLLIN) {
			fds[i].events |= POLL_EVENT_IN;
		}
		if (pfds[i].revents & POLLOUT) {
			fds[i].events |= POLL_EVENT_OUT;
		}
		if (pfds[i].revents & POLLERR) {
			fds[i].events |= POLL_EVENT_ERR;
		}
		if (pfds[i].revents & POLLHUP) {
			fds[i].events |= POLL_EVENT_HUP;
		}
		if (pfds[i].revents & POLLNVAL) {
			fds[i].events |= POLL_EVENT_NVAL;
		}
	}

	DDP_FREE_ARRAY(struct pollfd, pfds, nfds);
	return ret;
}
#else
static ddpint poll_linux(PollFd *fds, ddpint nfds, ddpint timeout) {
	struct pollfd *pfds = DDP_ALLOCATE(struct pollfd, nfds);

	for (int i = 0; i < nfds; i++) {
		pfds[i].fd = fds[i].sock.fd;
		pfds[i].events = 0;
		if (fds[i].events & POLL_EVENT_IN) {
			pfds[i].events |= POLLIN;
		}
		if (fds[i].events & POLL_EVENT_OUT) {
			pfds[i].events |= POLLOUT;
		}
		if (fds[i].events & POLL_EVENT_ERR) {
			pfds[i].events |= POLLERR;
		}
		if (fds[i].events & POLL_EVENT_HUP) {
			pfds[i].events |= POLLHUP;
		}
		if (fds[i].events & POLL_EVENT_NVAL) {
			pfds[i].events |= POLLNVAL;
		}
		pfds[i].revents = 0;
	}

	int ret;
	while (1) {
		ret = poll(pfds, nfds, timeout);
		if (ret < 0 && errno == EINTR) { // interrupted by signal, retry
			continue;
		} else if (ret < 0) {
			DDP_FREE_ARRAY(struct pollfd, pfds, nfds);
			if (errno == EINTR) {
				return POLL_TIMEOUT; // signal occured
			}

			ddp_error("poll Fehler: ", true);
			return POLL_ERR;
		} else if (ret == 0) {
			DDP_FREE_ARRAY(struct pollfd, pfds, nfds);
			return POLL_TIMEOUT;
		}

		for (int i = 0; i < nfds; i++) {
			fds[i].events = 0;
			if (pfds[i].revents & POLLIN) {
				fds[i].events |= POLL_EVENT_IN;
			}
			if (pfds[i].revents & POLLOUT) {
				fds[i].events |= POLL_EVENT_OUT;
			}
			if (pfds[i].revents & POLLERR) {
				fds[i].events |= POLL_EVENT_ERR;
			}
			if (pfds[i].revents & POLLHUP) {
				fds[i].events |= POLL_EVENT_HUP;
			}
			if (pfds[i].revents & POLLNVAL) {
				fds[i].events |= POLL_EVENT_NVAL;
			}
		}

		DDP_FREE_ARRAY(struct pollfd, pfds, nfds);
		return ret;
	}
}
#endif // DDPOS_WINDOWS

ddpint Socket_Poll(PollFdListref fds, ddpint timeout) {
	DDP_MIGHT_ERROR;
	DDP_NETWORK_MIGHT_TIMEOUT;
	ddpint result =
#ifdef DDPOS_WINDOWS
		poll_winsock(fds->arr, fds->len, timeout);
#else
		poll_linux(fds->arr, fds->len, timeout);
#endif
	if (result == POLL_TIMEOUT) {
		DDP_NETWORK_DID_TIMEOUT;
	}
	return result;
}
