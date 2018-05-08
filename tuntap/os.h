#pragma once

/*
 * Extract the "MACINTOSH" flag from the compiler.
 */
#if defined(__APPLE__)
#define UNIX
#define MACINTOSH
#endif

/*
 * Extract the "SUNOS" flag from the compiler.
 */
#if defined(sun)
#define UNIX
#define SUNOS
#endif

/*
 * Extract the "UNIX" flag from compiler.
 */

#ifdef __linux__
#define UNIX
#define LINUX
#endif

/*
 * Extract the "BSD" flag from compiler.
 */

#if defined(BSD) || defined(__FreeBSD__) || defined(__NetBSD__) || defined(__OpenBSD__)
#define BSD
#define UNIX
#endif

/*
 * Extract the "MSDOS" flag from the compiler.
 */
#ifdef __MSDOS__
#define MSDOS
#undef UNIX
#endif

/*
 * Extract the "WINDOWS" flag from the compiler.
 */
#if defined(_Windows) || defined(__WINDOWS__) || \
	defined(__WIN32__) || defined(WIN32) || \
defined(__WINNT__) || defined(__NT__) || \
defined(_WIN32) || defined(_WIN64)
#define WINDOWS
#ifdef _MSC_VER
#define MSV
#if defined(DEBUG) || defined(DEBUGTRACE)
#ifdef NDEBUG
#undef NDEBUG
#endif
#else
#ifndef NDEBUG
#define NDEBUG
#endif
#endif
#else
#undef MSV
#endif
#undef UNIX
#undef MSDOS
#endif

/*
 * Remove the WINDOWS flag when using MACINTOSH.
 */
#ifdef MACINTOSH
#undef WINDOWS
#endif

/*
 * Assume UNIX if not Windows, Macintosh or MSDOS.
 */
#if !defined(WINDOWS) && !defined(MACINTOSH) && !defined(MSDOS)
#define UNIX
#endif
