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
	"fmt"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/google/renameio"
	"github.com/nthnn/xbin25"

	"github.com/shamaton/msgpack/v2"
)

type SerializableX25fs struct {
	Version      uint32           `msgpack:"version"`
	RootDir      *SerializableDir `msgpack:"root"`
	InodeCounter uint64           `msgpack:"inode_counter"`
}

type SerializableNode struct {
	Type string            `msgpack:"type"`
	Dir  *SerializableDir  `msgpack:"directory,omitempty"`
	File *SerializableFile `msgpack:"file,omitempty"`
}

type SerializableDir struct {
	Attributes fuse.Attr                   `msgpack:"attr"`
	Children   map[string]SerializableNode `msgpack:"children"`
}

type SerializableFile struct {
	Attributes fuse.Attr `msgpack:"attr"`
	Data       []byte    `msgpack:"data"`
}

const X25FS_VERSION = 10000

func SaveData(xfs *X25fs, cfg *xbin25.XBin25Config, diskFile string) error {
	xfs.RootDir.Mux.RLock()
	defer xfs.RootDir.Mux.RUnlock()

	sxfs := &SerializableX25fs{
		RootDir:      SerializeDir(xfs.RootDir),
		InodeCounter: CurrentInodeCounter(),
		Version:      X25FS_VERSION,
	}

	buf, err := msgpack.Marshal(sxfs)
	if err != nil {
		return fmt.Errorf("msgpack encode failed: %w", err)
	}

	encrypted, err := cfg.Marshall(buf)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	if err := renameio.WriteFile(diskFile, encrypted, 0o600); err != nil {
		return fmt.Errorf("atomic write failed: %w", err)
	}

	return nil
}

func LoadData(cfg *xbin25.XBin25Config, diskFile string) (*X25fs, error) {
	encrypted, err := os.ReadFile(diskFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	decrypted, err := cfg.Unmarshall(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	msgpackBytes, ok := decrypted.([]byte)
	if !ok {
		return nil, fmt.Errorf("decrypted data is not []byte")
	}

	var sxfs SerializableX25fs
	err = msgpack.Unmarshal(msgpackBytes, &sxfs)
	if err != nil {
		return nil, fmt.Errorf("msgpack decode failed: %w", err)
	}

	if sxfs.Version != X25FS_VERSION {
		return nil, fmt.Errorf(
			"unmatched version: expecting %d, got %d",
			X25FS_VERSION,
			sxfs.Version,
		)
	}

	LoadInodeCounter(sxfs.InodeCounter)
	xfs := &X25fs{
		RootDir: DeserializeDir(sxfs.RootDir, cfg),
	}
	return xfs, nil
}

func SerializeDir(directory *Dir) *SerializableDir {
	directory.Mux.RLock()
	defer directory.Mux.RUnlock()

	sd := &SerializableDir{
		Attributes: directory.Attributes,
		Children:   make(map[string]SerializableNode),
	}

	for name, node := range directory.Children {
		switch n := node.(type) {
		case *Dir:
			sd.Children[name] = SerializableNode{
				Type: "dir",
				Dir:  SerializeDir(n),
			}
		case *File:
			sd.Children[name] = SerializableNode{
				Type: "file",
				File: &SerializableFile{
					Attributes: n.Attributes,
					Data:       n.Data,
				},
			}
		}
	}

	return sd
}

func DeserializeDir(sd *SerializableDir, cfg *xbin25.XBin25Config) *Dir {
	d := &Dir{
		Attributes: sd.Attributes,
		Children:   make(map[string]fs.Node),
		Config:     cfg,
	}

	for name, node := range sd.Children {
		switch node.Type {
		case "dir":
			d.Children[name] = DeserializeDir(node.Dir, cfg)

		case "file":
			d.Children[name] = &File{
				Attributes: node.File.Attributes,
				Data:       node.File.Data,
			}
		}
	}
	return d
}
