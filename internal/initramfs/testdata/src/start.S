;;; SPDX-FileCopyrightText: 2024 Tobias Böhm <code@aibor.de>
;;;
;;; SPDX-License-Identifier: MIT

/*
 * Custom _start entry function for ELF files for x86_64 machines. Needed if we 
 * do not want to use libc. Calls main() and exits wit the return value of
 * main. It does not set up argc/argv arguments for main().
 *
 * Taken from:
 *   https://www.reddit.com/r/C_Programming/comments/se3kgi/comment/huguya2/
 *
 * See also this for x86_64 version:
 *   https://github.com/LiveAsynchronousVisualizedArchitecture/lava/blob/704fbedec32b94d59a1dc14bc69611da2af215ba/tinylibc_test/C_without_stdlib.txt#L895
 *
 * And this for i386:
 *   https://github.com/LiveAsynchronousVisualizedArchitecture/lava/blob/704fbedec32b94d59a1dc14bc69611da2af215ba/tinylibc_test/C_without_stdlib.txt#L1298
 */

#define SYS_exit 60

	.globl _start
_start:
	call  main
	movl  %eax, %edi       // move exit code
	movl  $SYS_exit, %eax //set syscall
	syscall
