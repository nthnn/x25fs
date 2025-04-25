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
	"context"
	"sync"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type File struct {
	Mux        sync.RWMutex
	Attributes fuse.Attr
	Data       []byte
}

const MAX_FILE_SIZE = 536870912

func (file *File) Attr(
	ctx context.Context,
	attr *fuse.Attr,
) error {
	*attr = file.Attributes
	return nil
}

func (file *File) Setattr(
	ctx context.Context,
	req *fuse.SetattrRequest,
	res *fuse.SetattrResponse,
) error {
	file.Mux.Lock()
	defer file.Mux.Unlock()

	if req.Valid.Mode() {
		file.Attributes.Mode = req.Mode
	}

	if req.Valid.Uid() {
		file.Attributes.Uid = req.Uid
	}

	if req.Valid.Gid() {
		file.Attributes.Gid = req.Gid
	}

	if req.Valid.Atime() {
		file.Attributes.Atime = req.Atime
	}

	if req.Valid.Mtime() {
		file.Attributes.Mtime = req.Mtime
	}

	if req.Valid.Size() {
		currentSize := len(file.Data)
		newSize := int(req.Size)

		if newSize < currentSize {
			file.Data = file.Data[:newSize]
		} else if newSize > currentSize {
			if newSize > MAX_FILE_SIZE {
				return syscall.EFBIG
			}

			newData := make([]byte, newSize)
			copy(newData, file.Data)

			file.Data = newData
		}

		file.Attributes.Size = req.Size
		file.Attributes.Mtime = time.Now()
	}

	res.Attr = file.Attributes
	return nil
}

func (file *File) Getattr(
	ctx context.Context,
	req *fuse.GetattrRequest,
	resp *fuse.GetattrResponse,
) error {
	file.Mux.RLock()
	defer file.Mux.RUnlock()

	resp.Attr = file.Attributes
	return nil
}

func (file *File) Open(
	ctx context.Context,
	req *fuse.OpenRequest,
	res *fuse.OpenResponse,
) (fs.Handle, error) {
	return file, nil
}

func (file *File) Flush(
	ctx context.Context,
	req *fuse.FlushRequest,
) error {
	return nil
}

func (file *File) Release(
	ctx context.Context,
	req *fuse.ReleaseRequest,
) error {
	return nil
}

func (file *File) Read(
	ctx context.Context,
	req *fuse.ReadRequest,
	res *fuse.ReadResponse,
) error {
	file.Mux.RLock()
	defer file.Mux.RUnlock()

	if req.Offset >= int64(len(file.Data)) {
		res.Data = []byte{}
		return nil
	}

	offset := int(req.Offset)
	end := offset + req.Size

	if end > len(file.Data) {
		end = len(file.Data)
	}

	res.Data = file.Data[offset:end]
	return nil
}

func (file *File) Write(
	ctx context.Context,
	req *fuse.WriteRequest,
	res *fuse.WriteResponse,
) error {
	file.Mux.Lock()
	defer file.Mux.Unlock()

	reqLen := uint64(len(req.Data))
	newSize := uint64(req.Offset) + reqLen

	if newSize > MAX_FILE_SIZE {
		return syscall.EFBIG
	}

	fileLen := uint64(len(file.Data))
	if newSize > fileLen {
		newCapacity := newSize
		if newCapacity < fileLen*2 {
			newCapacity = fileLen * 2
		}

		if newCapacity > MAX_FILE_SIZE {
			newCapacity = MAX_FILE_SIZE
		}

		newData := make([]byte, newSize, newCapacity)
		copy(newData, file.Data)

		file.Data = newData
	} else if req.Offset+int64(reqLen) > int64(len(file.Data)) {
		file.Data = file.Data[:req.Offset+int64(reqLen)]
	}

	copy(file.Data[req.Offset:], req.Data)
	res.Size = int(reqLen)

	file.Attributes.Size = uint64(len(file.Data))
	file.Attributes.Mtime = time.Now()

	return nil
}
