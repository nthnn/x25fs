/*
 * Copyright 2025 Nathanne Isip
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	"github.com/nthnn/xbin25"
)

func CheckMountpointSecurity(mountpoint string) error {
	absPath, err := filepath.Abs(mountpoint)
	if err != nil {
		return fmt.Errorf("couldn't resolve absolute path: %w", err)
	}

	fi, err := os.Lstat(absPath)
	if err == nil && (fi.Mode()&os.ModeSymlink) != 0 {
		return fmt.Errorf("mountpoint cannot be a symlink")
	}

	parent := filepath.Dir(absPath)
	parentInfo, err := os.Stat(parent)

	if err != nil {
		return fmt.Errorf("couldn't stat parent directory: %w", err)
	}

	parentMode := parentInfo.Mode()
	if (parentMode&0002) != 0 && (parentMode&os.ModeSticky) == 0 {
		return fmt.Errorf("parent directory is world-writable without sticky bit")
	}

	return nil
}

func main() {
	var rlim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &rlim); err == nil {
		if MAX_FILE_SIZE > 0 && rlim.Cur > MAX_FILE_SIZE {
			rlim.Cur = MAX_FILE_SIZE
			_ = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &rlim)
		}
	}

	encCert := flag.String("encrypt-cert", "", "PEM file for encryption public key")
	encKey := flag.String("encrypt-key", "", "PEM file for decryption private key")
	signCert := flag.String("sign-cert", "", "PEM file for signature public key")
	signKey := flag.String("sign-key", "", "PEM file for signing private key")
	label := flag.String("label", "", "context label for OAEP encryption")
	dur := flag.Duration("duration", 36*time.Hour, "max age for replay protection")
	blockSize := flag.Int("block-size", 1024*1024, "compression block size")
	diskFile := flag.String("disk", "data.x25disk", "Path to the disk image file")

	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <mountpoint>\n", os.Args[0])
		os.Exit(1)
	}

	mountpoint := flag.Arg(0)
	if err := CheckMountpointSecurity(mountpoint); err != nil {
		log.Fatal("Mountpoint security check failed:", err)
	}

	diskPath, err := filepath.Abs(*diskFile)
	if err != nil {
		log.Fatal("Failed to get absolute path: ", err)
		return
	} else {
		log.Println("Persistent data will be saved on: ", diskPath)
	}

	cfg := xbin25.NewConfig(
		*encCert,
		*encKey,
		*signCert,
		*signKey,
		*label,
		*dur,
		*blockSize,
	)

	var xfs *X25fs
	if _, err := os.Stat(diskPath); err == nil {
		xfs, err = LoadData(cfg, diskPath)
		if err != nil {
			log.Fatal("Read disk: ", err)
		}
	} else {
		uid, gid := os.Getuid(), os.Getgid()
		if sudoUID := os.Getenv("SUDO_UID"); sudoUID != "" {
			if v, err := strconv.Atoi(sudoUID); err == nil {
				uid = v
			}
		}
		if sudoGID := os.Getenv("SUDO_GID"); sudoGID != "" {
			if v, err := strconv.Atoi(sudoGID); err == nil {
				gid = v
			}
		}

		xfs = &X25fs{
			RootDir: &Dir{
				Attributes: fuse.Attr{
					Inode: 1,
					Mode:  os.ModeDir | 0755,
					Uid:   uint32(uid),
					Gid:   uint32(gid),
				},
				Children: make(map[string]fs.Node),
				Config:   cfg,
			},
		}
	}

	conn, err := fuse.Mount(
		mountpoint,
		fuse.FSName("x25fs"),
		fuse.Subtype("x25fs"),
		fuse.DefaultPermissions(),
		fuse.AsyncRead(),
		fuse.CacheSymlinks(),
		fuse.WritebackCache(),
	)

	if err != nil {
		log.Fatal("Mount:", err)
	}

	connClosed := false
	defer func() {
		if !connClosed {
			conn.Close()
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	serveErr := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in fs.Serve: %v\n%s", r, debug.Stack())
				_ = fuse.Unmount(mountpoint)
			}
		}()

		serveErr <- fs.Serve(conn, xfs)
	}()

	select {
	case <-sigChan:
		log.Println("Received interrupt, unmounting...")
		if err := fuse.Unmount(mountpoint); err != nil {
			log.Fatal("Unmount error:", err)
		}

		log.Println("Waiting for filesystem to finish...")
		if err := <-serveErr; err != nil {
			log.Println("Serve error after unmount:", err)
		}

	case err := <-serveErr:
		if err != nil {
			log.Fatal("Serve error:", err)
		}
	}

	saveErr := SaveData(xfs, cfg, diskPath)
	log.Println("Saving filesystem state...")

	if saveErr != nil {
		log.Printf("Save failed: %v", saveErr)
	} else {
		log.Println("Successfully saved to", diskPath)
	}

	conn.Close()
	connClosed = true

	if saveErr != nil {
		os.Exit(1)
	}
}
