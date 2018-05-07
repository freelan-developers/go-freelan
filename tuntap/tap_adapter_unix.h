#pragma once

#include <string.h>
#include <errno.h>
#include <unistd.h>

#include <sys/types.h>
#include <sys/socket.h>
#include <sys/ioctl.h>

#include <net/if.h>
#include <netinet/ip.h>

int set_ipv4_address(const char* name, char ipv4_address[4], int prefix_length) {
    int fd;

    fd = socket(AF_INET, SOCK_DGRAM, 0);

    if (fd < 0) {
        return errno;
    }

    struct ifreq ifr = {};

    strncpy(ifr.ifr_name, name, IFNAMSIZ - 1);

    struct sockaddr_in* ifr_sockaddr = (struct sockaddr_in*)(&ifr.ifr_addr);
    ifr_sockaddr->sin_family = AF_INET;
    ifr_sockaddr->sin_len = sizeof(struct sockaddr_in);
    memcpy(&ifr_sockaddr->sin_addr.s_addr, &ipv4_address[0], 4);

    if (ioctl(fd, SIOCSIFADDR, &ifr) < 0) {
        // If the error is that the address is already set, we just continue.
        if (errno != EEXIST) {
            close(fd);
            return errno;
        }
    }

    if (prefix_length > 0) {
        ifr_sockaddr->sin_addr.s_addr = htonl((0xFFFFFFFF >> (32 - prefix_length)) << (32 - prefix_length));

        if (ioctl(fd, SIOCSIFNETMASK, &ifr) < 0) {
            // If the error is that the address is already set, we just continue.
            if (errno != EEXIST) {
                close(fd);
                return errno;
            }
        }
    }

    close(fd);

    return 0;
}
