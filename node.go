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

import "sync/atomic"

var inodeCounter uint64 = 2

func GetInodeAndIncrease() uint64 {
	new := atomic.AddUint64(&inodeCounter, 1)
	if new == 0 {
		panic("inode counter overflow")
	}

	return new
}

func LoadInodeCounter(saved uint64) {
	atomic.StoreUint64(&inodeCounter, saved)
}

func CurrentInodeCounter() uint64 {
	return atomic.LoadUint64(&inodeCounter)
}

func GetTotalInodes(dir *Dir) uint64 {
	dir.Mux.RLock()
	defer dir.Mux.RUnlock()

	count := uint64(1)
	for _, node := range dir.Children {
		switch n := node.(type) {
		case *Dir:
			count += GetTotalInodes(n)
		case *File:
			count++
		}
	}

	return count
}
