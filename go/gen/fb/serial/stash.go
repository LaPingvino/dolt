// Copyright 2022-2023 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package serial

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type Stash struct {
	_tab flatbuffers.Table
}

func InitStashRoot(o *Stash, buf []byte, offset flatbuffers.UOffsetT) error {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	o.Init(buf, n+offset)
	if StashNumFields < o.Table().NumFields() {
		return flatbuffers.ErrTableHasUnknownFields
	}
	return nil
}

func TryGetRootAsStash(buf []byte, offset flatbuffers.UOffsetT) (*Stash, error) {
	x := &Stash{}
	return x, InitStashRoot(x, buf, offset)
}

func GetRootAsStash(buf []byte, offset flatbuffers.UOffsetT) *Stash {
	x := &Stash{}
	InitStashRoot(x, buf, offset)
	return x
}

func TryGetSizePrefixedRootAsStash(buf []byte, offset flatbuffers.UOffsetT) (*Stash, error) {
	x := &Stash{}
	return x, InitStashRoot(x, buf, offset+flatbuffers.SizeUint32)
}

func GetSizePrefixedRootAsStash(buf []byte, offset flatbuffers.UOffsetT) *Stash {
	x := &Stash{}
	InitStashRoot(x, buf, offset+flatbuffers.SizeUint32)
	return x
}

func (rcv *Stash) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Stash) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *Stash) StashRootAddr(j int) byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.GetByte(a + flatbuffers.UOffsetT(j*1))
	}
	return 0
}

func (rcv *Stash) StashRootAddrLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func (rcv *Stash) StashRootAddrBytes() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Stash) MutateStashRootAddr(j int, n byte) bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.MutateByte(a+flatbuffers.UOffsetT(j*1), n)
	}
	return false
}

func (rcv *Stash) HeadCommitAddr(j int) byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.GetByte(a + flatbuffers.UOffsetT(j*1))
	}
	return 0
}

func (rcv *Stash) HeadCommitAddrLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func (rcv *Stash) HeadCommitAddrBytes() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Stash) MutateHeadCommitAddr(j int, n byte) bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		a := rcv._tab.Vector(o)
		return rcv._tab.MutateByte(a+flatbuffers.UOffsetT(j*1), n)
	}
	return false
}

func (rcv *Stash) BranchName() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Stash) Desc() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

const StashNumFields = 4

func StashStart(builder *flatbuffers.Builder) {
	builder.StartObject(StashNumFields)
}
func StashAddStashRootAddr(builder *flatbuffers.Builder, stashRootAddr flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(stashRootAddr), 0)
}
func StashStartStashRootAddrVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(1, numElems, 1)
}
func StashAddHeadCommitAddr(builder *flatbuffers.Builder, headCommitAddr flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(1, flatbuffers.UOffsetT(headCommitAddr), 0)
}
func StashStartHeadCommitAddrVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(1, numElems, 1)
}
func StashAddBranchName(builder *flatbuffers.Builder, branchName flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(2, flatbuffers.UOffsetT(branchName), 0)
}
func StashAddDesc(builder *flatbuffers.Builder, desc flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(3, flatbuffers.UOffsetT(desc), 0)
}
func StashEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
