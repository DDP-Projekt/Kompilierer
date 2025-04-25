#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/ddpwindows.h"
#include "DDP/error.h"
#include <assert.h>
#include <string.h>

#ifdef DDPOS_WINDOWS
#include <winsock2.h>
#include <ws2tcpip.h>
#else
#include <arpa/inet.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/types.h>
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

#endif // DDPOS_WINDOWS

struct addrinfo *Lade_AddressInfo(int family, int type, const ddpstring *name, const ddpstring *service) {
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
		ddp_error("inet_ntop Fehler: ", true);
		*ret = DDP_EMPTY_STRING;
		return;
	}

	ddp_string_from_constant(ret, ipstr);
}

void Socket_Erstellen(ddpsocket *ret, int family, int type) {
	int sock = socket(family, type, 0);
	if (sock < 0) {
		ddp_error("socket Fehler: ", true);
	}
	ret->fd = sock;
	ret->family = family;
	ret->type = type;
}

void Socket_Binden(const ddpsocket *sock, struct addrinfo *info) {
	for (struct addrinfo *p = info; p != NULL; p = p->ai_next) {
		if (bind(sock->fd, info->ai_addr, info->ai_addrlen) < 0) {
			ddp_error("bind Fehler: ", true);
		} else {
			break;
		}
	}
}

void Socket_Verbinden(const ddpsocket *sock, struct addrinfo *info) {
	for (struct addrinfo *p = info; p != NULL; p = p->ai_next) {
		if (connect(sock->fd, info->ai_addr, info->ai_addrlen) < 0) {
			ddp_error("connect Fehler: ", true);
		} else {
			break;
		}
	}
}

void Socket_Zuhoeren(const ddpsocket *sock, ddpint backlog) {
	if (listen(sock->fd, (int)backlog) < 0) {
		ddp_error("listen Fehler: ", true);
	}
}

void Socket_Verbindung_Annehmen(ddpsocket *ret, const ddpsocket *sock, ddpsockaddr_storage *client_addr) {
	client_addr->size = sizeof client_addr->storage;
	int conn = accept(sock->fd, (struct sockaddr *)&client_addr->storage, &client_addr->size);
	if (conn < 0) {
		ddp_error("accept Fehler: ", true);
	}
	ret->fd = conn;
	ret->family = ((struct sockaddr_storage *)client_addr)->ss_family;
	ret->type = sock->type;
}

ddpint Socket_Senden(const ddpsocket *sock, const ddpstringref data) {
	ddpint result = 0;
	while (result < data->cap - 1) {
		int sent;
		if ((sent = send(sock->fd, &data->str[result], data->cap - 1 - result, 0)) < 0) {
			ddp_error("send Fehler: ", true);
			return result;
		}
		result += sent;
	}
	return result;
}

void Socket_Empfangen(ddpstring *ret, ddpsocket *sock, const ddpint max) {
	char *buf = DDP_ALLOCATE(char, max + 1);
	ddpint got = (ddpint)recv(sock->fd, buf, max, 0);
	if (got < 0) {
		*ret = DDP_EMPTY_STRING;
		DDP_FREE_ARRAY(char, buf, max + 1);
		ddp_error("recv Fehler: ", true);
		return;
	}

	if (got == 0) {
		*ret = DDP_EMPTY_STRING;
		DDP_FREE_ARRAY(char, buf, max + 1);
		return;
	}

	ret->str = buf;
	ret->cap = got + 1;
	ddp_reallocate(buf, max + 1, ret->cap);
	ret->str[got] = '\0';
}

ddpint Socket_Senden_An_Klient(const ddpsocket *sock, const ddpstring *data, const ddpsockaddr_storage *client) {
	ddpint sent = 0;
	if ((sent = (ddpint)sendto(sock->fd, data->str, data->cap - 1, 0, (struct sockaddr *)&client->storage, client->size)) < 0) {
		ddp_error("send Fehler: ", true);
	}

	return sent;
}

ddpint Socket_Senden_An(const ddpsocket *sock, const ddpstring *data, const struct addrinfo *info) {
	ddpsockaddr_storage client;
	client.size = info->ai_addrlen;
	memcpy(&client.storage, info->ai_addr, info->ai_addrlen);
	return Socket_Senden_An_Klient(sock, data, &client);
}

void Socket_Empfangen_Von(ddpstring *ret, ddpsocket *sock, ddpint max, ddpsockaddr_storage *client_addr) {
	char *buf = DDP_ALLOCATE(char, max + 1);
	client_addr->size = sizeof(client_addr->storage);
	ddpint got = (ddpint)recvfrom(sock->fd, buf, max, 0, (struct sockaddr *)&client_addr->storage, &client_addr->size);
	if (got < 0) {
		*ret = DDP_EMPTY_STRING;
		DDP_FREE_ARRAY(char, buf, max + 1);
		ddp_error("recv Fehler: ", true);
		return;
	}
	ret->str = buf;
	ret->cap = got + 1;
	ddp_reallocate(buf, max + 1, ret->cap);
	ret->str[got] = '\0';
}
