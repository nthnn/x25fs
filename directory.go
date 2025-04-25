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
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/nthnn/xbin25"
)

type Dir struct {
	Mux        sync.RWMutex
	Attributes fuse.Attr
	Children   map[string]fs.Node
	Config     *xbin25.XBin25Config
}

func (directory *Dir) Attr(
	ctx context.Context,
	attr *fuse.Attr,
) error {
	*attr = directory.Attributes
	return nil
}

func hasWritePermission(uid, gid uint32, attr fuse.Attr) bool {
	if uid == attr.Uid {
		return attr.Mode&0200 != 0
	} else if gid == attr.Gid {
		return attr.Mode&0020 != 0
	} else {
		return attr.Mode&0002 != 0
	}
}

func (directory *Dir) Setattr(
	ctx context.Context,
	req *fuse.SetattrRequest,
	resp *fuse.SetattrResponse,
) error {
	directory.Mux.RLock()
	defer directory.Mux.RUnlock()

	if req.Valid.Mode() {
		directory.Attributes.Mode = req.Mode
	}

	if req.Valid.Uid() {
		directory.Attributes.Uid = req.Uid
	}

	if req.Valid.Gid() {
		directory.Attributes.Gid = req.Gid
	}

	if req.Valid.Atime() {
		directory.Attributes.Atime = req.Atime
	}

	if req.Valid.Mtime() {
		directory.Attributes.Mtime = req.Mtime
	}

	resp.Attr = directory.Attributes
	return nil
}

func (directory *Dir) Getattr(
	ctx context.Context,
	req *fuse.GetattrRequest,
	resp *fuse.GetattrResponse,
) error {
	directory.Mux.RLock()
	defer directory.Mux.RUnlock()

	resp.Attr = directory.Attributes
	return nil
}

func (directory *Dir) Lookup(
	ctx context.Context,
	name string,
) (fs.Node, error) {
	directory.Mux.RLock()
	defer directory.Mux.RUnlock()

	if child, ok := directory.Children[name]; ok {
		return child, nil
	}

	return nil, syscall.ENOENT
}

func (directory *Dir) Create(
	ctx context.Context,
	req *fuse.CreateRequest,
	res *fuse.CreateResponse,
) (fs.Node, fs.Handle, error) {
	directory.Mux.Lock()
	defer directory.Mux.Unlock()

	name := req.Name
	if err := ValidateName(name); err != nil {
		return nil, nil, err
	}

	if _, exists := directory.Children[name]; exists {
		return nil, nil, syscall.EEXIST
	}

	uid, gid := req.Header.Uid, req.Header.Gid
	fileMode := directory.Attributes.Mode

	if uid == directory.Attributes.Uid {
		if fileMode&0200 == 0 {
			return nil, nil, syscall.EACCES
		}
	} else if gid == directory.Attributes.Gid {
		if fileMode&0020 == 0 {
			return nil, nil, syscall.EACCES
		}
	} else {
		if fileMode&0002 == 0 {
			return nil, nil, syscall.EACCES
		}
	}

	if uid == directory.Attributes.Uid {
		if fileMode&0100 == 0 {
			return nil, nil, syscall.EACCES
		}
	} else if gid == directory.Attributes.Gid {
		if fileMode&0010 == 0 {
			return nil, nil, syscall.EACCES
		}
	} else {
		if fileMode&0001 == 0 {
			return nil, nil, syscall.EACCES
		}
	}

	now := time.Now()
	file := &File{
		Attributes: fuse.Attr{
			Inode: GetInodeAndIncrease(),
			Mode:  req.Mode.Perm(),
			Size:  0,
			Atime: now,
			Mtime: now,
			Ctime: now,
			Uid:   req.Header.Uid,
			Gid:   req.Header.Gid,
		},
		Data: []byte{},
	}

	directory.Children[name] = file
	directory.Attributes.Mtime = now
	directory.Attributes.Ctime = now

	return file, file, nil
}

func (directory *Dir) Mkdir(
	ctx context.Context,
	req *fuse.MkdirRequest,
) (fs.Node, error) {
	directory.Mux.Lock()
	defer directory.Mux.Unlock()

	name := req.Name
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	if _, exists := directory.Children[name]; exists {
		return nil, syscall.EEXIST
	}

	if req.Header.Uid != directory.Attributes.Uid {
		return nil, syscall.EPERM
	}

	if req.Header.Gid != directory.Attributes.Gid {
		return nil, syscall.EPERM
	}

	now := time.Now()
	safePerm := uint32(req.Mode.Perm()) & 0o777

	dir := &Dir{
		Attributes: fuse.Attr{
			Inode: GetInodeAndIncrease(),
			Mode:  os.ModeDir | os.FileMode(safePerm),
			Atime: now,
			Mtime: now,
			Ctime: now,
			Uid:   req.Header.Uid,
			Gid:   req.Header.Gid,
		},
		Children: make(map[string]fs.Node),
		Config:   directory.Config,
	}

	directory.Children[name] = dir
	directory.Attributes.Ctime = now
	directory.Attributes.Mtime = now

	return dir, nil
}

func (directory *Dir) ReadDirAll(
	ctx context.Context,
) ([]fuse.Dirent, error) {
	directory.Mux.RLock()
	defer directory.Mux.RUnlock()

	var entries []fuse.Dirent
	for name, node := range directory.Children {
		var typ fuse.DirentType

		switch node.(type) {
		case *Dir:
			typ = fuse.DT_Dir
		case *File:
			typ = fuse.DT_File
		}

		entries = append(entries, fuse.Dirent{
			Inode: node.(AttrGetter).GetAttr().Inode,
			Name:  name,
			Type:  typ,
		})
	}

	return entries, nil
}

func (directory *Dir) Remove(
	ctx context.Context,
	req *fuse.RemoveRequest,
) error {
	directory.Mux.Lock()
	defer directory.Mux.Unlock()

	name := req.Name
	if err := ValidateName(name); err != nil {
		return err
	}

	if _, exists := directory.Children[name]; !exists {
		return syscall.ENOENT
	}

	if req.Header.Uid != directory.Attributes.Uid {
		return syscall.EPERM
	}

	if req.Header.Gid != directory.Attributes.Gid {
		return syscall.EPERM
	}

	delete(directory.Children, name)
	directory.Attributes.Mtime = time.Now()
	directory.Attributes.Ctime = time.Now()

	return nil
}

func (directory *Dir) Rename(
	ctx context.Context,
	req *fuse.RenameRequest,
	newDir fs.Node,
) error {
	if req.NewName != filepath.Base(req.NewName) ||
		req.NewName == ".." || req.NewName == "." {
		return syscall.EINVAL
	}

	if req.OldName != filepath.Base(req.OldName) ||
		req.OldName == ".." || req.OldName == "." {
		return syscall.EINVAL
	}

	targetDir, ok := newDir.(*Dir)
	if !ok {
		return syscall.EINVAL
	}

	if !hasWritePermission(
		req.Header.Uid,
		req.Header.Gid,
		directory.Attributes) ||
		!hasWritePermission(
			req.Header.Uid,
			req.Header.Gid,
			targetDir.Attributes,
		) {
		return syscall.EACCES
	}

	first, second := directory, targetDir
	if directory.Attributes.Inode > targetDir.Attributes.Inode {
		first, second = targetDir, directory
	}

	first.Mux.Lock()
	defer first.Mux.Unlock()
	if first != second {
		second.Mux.Lock()
		defer second.Mux.Unlock()
	}

	child, exists := directory.Children[req.OldName]
	if !exists {
		return syscall.ENOENT
	}

	delete(targetDir.Children, req.NewName)
	delete(directory.Children, req.OldName)
	targetDir.Children[req.NewName] = child

	now := time.Now()
	directory.Attributes.Mtime = now
	directory.Attributes.Ctime = now

	targetDir.Attributes.Mtime = now
	targetDir.Attributes.Ctime = now

	return nil
}
