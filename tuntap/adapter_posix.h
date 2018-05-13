#pragma once

#include "os.h"

#include <assert.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <ifaddrs.h>
#include <errno.h>
#include <unistd.h>
#include <fcntl.h>
#include <netinet/in.h>

#ifdef LINUX

#include <linux/if_tun.h>
#include <sys/sysmacros.h>
#include <net/if.h>

/**
 * \struct in6_ifreq
 * \brief Replacement structure since the include of linux/ipv6.h introduces conflicts.
 *
 * If someone comes up with a better solution, feel free to contribute :)
 */
struct in6_ifreq
{
    struct in6_addr ifr6_addr; /**< IPv6 address */
    uint32_t ifr6_prefixlen; /**< Length of the prefix */
    int ifr6_ifindex; /**< Interface index */
};

#elif defined(MACINTOSH) || defined(BSD)

/*
 * Note for Mac OS X users : you have to download and install the tun/tap driver from (http://tuntaposx.sourceforge.net).
 */
#ifndef __NetBSD__
#include <net/if_var.h>
#endif

#include <net/if_types.h>
#include <net/if_dl.h>
#include <net/if.h>
#include <netinet/in.h>
#include <netinet6/in6_var.h>

#endif

typedef enum {
    AL_ETHERNET = 0,
    AL_IP = 1,
} adapter_layer;

struct adapter {
    int fd;
    char name[IFNAMSIZ];
};

const struct adapter* open_adapter(adapter_layer layer, const char* _name) {
#if defined(LINUX)
    const char* dev_name = "/dev/net/tap";

    if (layer == AL_IP) {
        dev_name = "/dev/net/tun";
    }

    if (access(dev_name, F_OK) == -1) {
        if (errno != ENOENT) {
            return NULL;
        }

        // No tap found, create one.
        if (mknod(dev_name, S_IFCHR | S_IRUSR | S_IWUSR, makedev(10, 200)) == -1) {
            return NULL;
        }
    }

    int device = open(dev_name, O_RDWR);

    if (device < 0) {
        return NULL;
    }

    struct ifreq ifr = {};

    {
        ifr.ifr_flags = IFF_NO_PI;

#if defined(IFF_ONE_QUEUE) && defined(SIOCSIFTXQLEN)
        ifr.ifr_flags |= IFF_ONE_QUEUE;
#endif

        if (layer == AL_ETHERNET) {
            ifr.ifr_flags |= IFF_TAP;
        } else {
            ifr.ifr_flags |= IFF_TUN;
        }

        if (_name != NULL) {
            strncpy(ifr.ifr_name, _name, IFNAMSIZ);
        }

        // Set the parameters on the tun device.
        if (ioctl(device, TUNSETIFF, (void *)&ifr) < 0) {
            close(device);
            return NULL;
        }
    }

#if defined(IFF_ONE_QUEUE) && defined(SIOCSIFTXQLEN)
    int sock = socket(AF_INET, SOCK_DGRAM, 0);

    if (sock < 0) {
        return NULL;
    }

    {
        ifr.ifr_flags = 0;
        ifr.ifr_qlen = 100; // 100 is the default value

        if (getuid() == 0 && ioctl(sock, SIOCSIFTXQLEN, (void *)&ifr) < 0) {
            close(sock);
            return NULL;
        }
    }
#endif /* IFF_ONE_QUEUE */

    struct adapter* result = (struct adapter*)malloc(sizeof(struct adapter));
    result->fd = device;
    strncpy(result->name, ifr.ifr_name, IFNAMSIZ);

#else /* *BSD and Mac OS X */
    const char* dev_type = "tap";

    if (layer == AL_IP) {
        dev_type = "tun";
    }

    int device = -1;
    char path[256] = {};

    if (_name != NULL) {
        if (snprintf(path, 256, "/dev/%s", _name) < 0) {
            errno = EINVAL;
            return NULL;
        }

        device = open(path, O_RDWR);
    } else {
        for (unsigned int i = 0 ; device < 0; ++i) {

            if (snprintf(path, 256, "/dev/%s%u", dev_type, i) < 0) {
                errno = EINVAL;
                return NULL;
            }

            _name = &path[5];

            device = open(path, O_RDWR);

            if ((device < 0) && (errno == ENOENT)) {
                // We reached the end of the available tap adapters.
                break;
            }
        }
    }

    if (device < 0) {
        errno = ENOENT;
        return NULL;
    }

    struct stat st = {};

    if (fstat(device, &st) != 0) {
        return NULL;
    }

    struct adapter* result = (struct adapter*)malloc(sizeof(struct adapter));
    result->fd = device;

#ifdef __NetBSD__
    if (devname_r(st.st_dev, S_IFCHR, result->name, IFNAMSIZ - 1) != 0) {
        strncpy(result->name, _name, IFNAMSIZ);
    }
#elif defined(__OpenBSD__)
    char* n = devname(st.st_dev, S_IFCHR);

    if (n) {
        strncpy(result->name, n, IFNAMSIZ - 1);
    } else {
        strncpy(result->name, _name, IFNAMSIZ);
    }
#else
    if (devname_r(st.st_dev, S_IFCHR, result->name, IFNAMSIZ - 1) == NULL) {
        strncpy(result->name, _name, IFNAMSIZ);
    }
#endif

#endif /* BSD & OSX */

    return result;
}

int close_adapter(struct adapter* ta) {
    // only attempt to destroy interface if non-root.
    if (getuid() == 0) {
#if defined(MACINTOSH) || defined(BSD)
        int sock = socket(AF_INET, SOCK_DGRAM, 0);

        if (sock < 0) {
            free(ta);
            return -1;
        }

        struct ifreq ifr = {};
        memset(ifr.ifr_name, 0x00, IFNAMSIZ);
        strncpy(ifr.ifr_name, ta->name, IFNAMSIZ);

        // Destroy the virtual tap device
        if (ioctl(sock, SIOCIFDESTROY, &ifr) < 0) {
            close(sock);
            free(ta);
            return -1;
        }

        close(sock);
#endif
    }

    if (close(ta->fd) != 0) {
        free(ta);
        return -1;
    }

    free(ta);
    return 0;
}

int set_adapter_connected_state(struct adapter* ta, int connected) {
    // as non-root, assume that existing TAP is correctly configured
    if (getuid() != 0) {
        return 0;
    }

    struct ifreq ifr = {};
    strncpy(ifr.ifr_name, ta->name, IFNAMSIZ);

    int sock = socket(AF_INET, SOCK_DGRAM, 0);

    // Get the interface flags
    if (ioctl(sock, SIOCGIFFLAGS, &ifr) < 0) {
        close(sock);
        return -1;
    }

    if (connected != 0) {
#ifdef MACINTOSH
        ifr.ifr_flags |= IFF_UP;
#else
        ifr.ifr_flags |= (IFF_UP | IFF_RUNNING);
#endif
    } else {
#ifdef MACINTOSH
        // Mac OS X: set_connected_state(false) seems to confuse the TAP
        // so do nothing for the moment.
        close(sock);
        return 0;
#else
        ifr.ifr_flags &= ~(IFF_UP | IFF_RUNNING);
#endif
    }

    // Set the interface UP
    if (ioctl(sock, SIOCSIFFLAGS, &ifr) < 0) {
        close(sock);
        return -1;
    }

    close(sock);

    return 0;
}

int set_adapter_mtu(struct adapter* ta, size_t _mtu) {
    struct ifreq ifr = {};
    strncpy(ifr.ifr_name, ta->name, IFNAMSIZ);

    int sock = socket(AF_INET, SOCK_DGRAM, 0);

    ifr.ifr_mtu = _mtu;

    if (ioctl(sock, SIOCSIFMTU, &ifr) < 0) {
        close(sock);
        return -1;
    }

    close(sock);
    return 0;
}

int set_adapter_ipv4(struct adapter* ta, struct in_addr addr, int prefixlen) {
    assert(prefixlen < 32);

    int sock = socket(AF_INET, SOCK_DGRAM, 0);

    struct ifreq ifr = {};
    strncpy(ifr.ifr_name, ta->name, IFNAMSIZ);

    struct sockaddr_in* ifr_a = (struct sockaddr_in*)(&ifr.ifr_addr);
    ifr_a->sin_family = AF_INET;
#ifdef BSD
    ifr_a->sin_len = sizeof(sockaddr_in);
#endif
    memcpy(&ifr_a->sin_addr.s_addr, &addr.s_addr, sizeof(struct in_addr));

    if (ioctl(sock, SIOCSIFADDR, &ifr) < 0) {
        // If the address is already set, we ignore it.
        if (errno != EEXIST) {
            close(sock);
            return -1;
        }
    }

    if (prefixlen > 0) {
        ifr_a->sin_addr.s_addr = htonl((0xFFFFFFFF >> (32 - prefixlen)) << (32 - prefixlen));

        if (ioctl(sock, SIOCSIFNETMASK, &ifr) < 0) {
            // If the mask is already set, we ignore it.
            if (errno != EEXIST) {
                close(sock);
                return -1;
            }
        }
    }

    close(sock);
    return 0;
}

int set_adapter_ipv6(struct adapter* ta, struct in6_addr addr, int prefixlen) {
    assert(prefixlen < 128);

    int sock = socket(AF_INET6, SOCK_DGRAM, 0);

#ifdef LINUX
    const unsigned int if_index = if_nametoindex(ta->name);

    if (if_index == 0) {
        close(sock);
        return -1;
    }

    struct in6_ifreq ifr = {};
    memcpy(&ifr.ifr6_addr.s6_addr, &addr.s6_addr, sizeof(struct in6_addr));
    ifr.ifr6_prefixlen = prefixlen;
    ifr.ifr6_ifindex = if_index;

    if (ioctl(sock, SIOCSIFADDR, &ifr) < 0)
#else
    struct in6_aliasreq iar = {};
    strncpy(iar.ifra_name, ta->name, IFNAMSIZ);

    ((struct sockaddr_in6*)(&iar.ifra_addr))->sin6_family = AF_INET6;
    ((struct sockaddr_in6*)(&iar.ifra_prefixmask))->sin6_family = AF_INET6;
    memcpy(&((struct sockaddr_in6*)(&iar.ifra_addr))->sin6_addr.s6_addr, &addr.s6_addr, sizeof(struct in6_addr));
    memset(((struct sockaddr_in6*)(&iar.ifra_prefixmask))->sin6_addr.s6_addr, 0xFF, prefixlen / 8);
    ((struct sockaddr_in6*)(&iar.ifra_prefixmask))->sin6_addr.s6_addr[prefixlen / 8] = (0xFF << (8 - (prefixlen % 8)));
    iar.ifra_lifetime.ia6t_pltime = 0xFFFFFFFF;
    iar.ifra_lifetime.ia6t_vltime = 0xFFFFFFFF;

#ifdef SIN6_LEN
    ((struct sockaddr_in6*)(&iar.ifra_addr))->sin6_len = sizeof(struct sockaddr_in6);
    ((struct sockaddr_in6*)(&iar.ifra_prefixmask))->sin6_len = sizeof(struct sockaddr_in6);
#endif

    if (ioctl(sock, SIOCAIFADDR_IN6, &iar) < 0)
#endif
    {
        // If the address is already set, we ignore it.
        if (errno != EEXIST) {
            close(sock);
            return -1;
        }
    }

    close(sock);
    return 0;
}

int set_adapter_remote_ipv4(struct adapter* ta, struct in_addr addr) {
#ifdef MACINTOSH
    // The TUN adapter for Mac OSX has a weird behavior regarding routes and ioctl.

    // For some reason, on Mac, setting up the IP address using ioctl() doesn't work for TUN devices.
    // OSX apparently does not create a route even though ifconfig indicates that the netmask is understood.
    // We must create it ourselves.
    errno = ENOSYS;
    return -1;
#else
    int sock = socket(AF_INET, SOCK_DGRAM, 0);

    struct ifreq ifr = {};
    strncpy(ifr.ifr_name, ta->name, IFNAMSIZ);

    struct sockaddr_in* ifr_dst_addr = (struct sockaddr_in*)(&ifr.ifr_dstaddr);
    ifr_dst_addr->sin_family = AF_INET;
#ifdef BSD
    ifr_dst_addr->sin_len = sizeof(struct sockaddr_in);
#endif
    memcpy(&ifr_dst_addr->sin_addr.s_addr, &addr.s_addr, sizeof(struct in_addr));

    if (ioctl(sock, SIOCSIFDSTADDR, &ifr) < 0) {
        // If the address is already set, we ignore it.
        if (errno != EEXIST) {
            close(sock);
            return -1;
        }
    }

    close(sock);
    return 0;
#endif
}
