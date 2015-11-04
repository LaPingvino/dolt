// This file was generated by nomdl/codegen.

package common

import (
	"github.com/attic-labs/noms/chunks"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/types"
)

var __commonPackageInFile_quad_tree_CachedRef ref.Ref

// This function builds up a Noms value that describes the type
// package implemented by this file and registers it with the global
// type package definition cache.
func init() {
	p := types.NewPackage([]types.TypeRef{
		types.MakeStructTypeRef("Node",
			[]types.Field{
				types.Field{"Geoposition", types.MakeTypeRef(ref.Parse("sha1-6d5e1c54214264058be9f61f4b4ece0368c8c678"), 0), false},
				types.Field{"Reference", types.MakeCompoundTypeRef(types.RefKind, types.MakePrimitiveTypeRef(types.ValueKind)), false},
			},
			types.Choices{},
		),
		types.MakeStructTypeRef("QuadTree",
			[]types.Field{
				types.Field{"Nodes", types.MakeCompoundTypeRef(types.ListKind, types.MakeTypeRef(ref.Ref{}, 0)), false},
				types.Field{"Tiles", types.MakeCompoundTypeRef(types.MapKind, types.MakePrimitiveTypeRef(types.StringKind), types.MakeTypeRef(ref.Ref{}, 1)), false},
				types.Field{"Depth", types.MakePrimitiveTypeRef(types.UInt8Kind), false},
				types.Field{"NumDescendents", types.MakePrimitiveTypeRef(types.UInt32Kind), false},
				types.Field{"Path", types.MakePrimitiveTypeRef(types.StringKind), false},
				types.Field{"Georectangle", types.MakeTypeRef(ref.Parse("sha1-6d5e1c54214264058be9f61f4b4ece0368c8c678"), 1), false},
			},
			types.Choices{},
		),
		types.MakeStructTypeRef("SQuadTree",
			[]types.Field{
				types.Field{"Nodes", types.MakeCompoundTypeRef(types.ListKind, types.MakeCompoundTypeRef(types.RefKind, types.MakePrimitiveTypeRef(types.ValueKind))), false},
				types.Field{"Tiles", types.MakeCompoundTypeRef(types.MapKind, types.MakePrimitiveTypeRef(types.StringKind), types.MakeCompoundTypeRef(types.RefKind, types.MakeTypeRef(ref.Ref{}, 2))), false},
				types.Field{"Depth", types.MakePrimitiveTypeRef(types.UInt8Kind), false},
				types.Field{"NumDescendents", types.MakePrimitiveTypeRef(types.UInt32Kind), false},
				types.Field{"Path", types.MakePrimitiveTypeRef(types.StringKind), false},
				types.Field{"Georectangle", types.MakeTypeRef(ref.Parse("sha1-6d5e1c54214264058be9f61f4b4ece0368c8c678"), 1), false},
			},
			types.Choices{},
		),
	}, []ref.Ref{
		ref.Parse("sha1-6d5e1c54214264058be9f61f4b4ece0368c8c678"),
	})
	__commonPackageInFile_quad_tree_CachedRef = types.RegisterPackage(&p)
}

// Node

type Node struct {
	_Geoposition Geoposition
	_Reference   RefOfValue

	ref *ref.Ref
}

func NewNode() Node {
	return Node{
		_Geoposition: NewGeoposition(),
		_Reference:   NewRefOfValue(ref.Ref{}),

		ref: &ref.Ref{},
	}
}

type NodeDef struct {
	Geoposition GeopositionDef
	Reference   ref.Ref
}

func (def NodeDef) New() Node {
	return Node{
		_Geoposition: def.Geoposition.New(),
		_Reference:   NewRefOfValue(def.Reference),
		ref:          &ref.Ref{},
	}
}

func (s Node) Def() (d NodeDef) {
	d.Geoposition = s._Geoposition.Def()
	d.Reference = s._Reference.TargetRef()
	return
}

var __typeRefForNode types.TypeRef

func (m Node) TypeRef() types.TypeRef {
	return __typeRefForNode
}

func init() {
	__typeRefForNode = types.MakeTypeRef(__commonPackageInFile_quad_tree_CachedRef, 0)
	types.RegisterStruct(__typeRefForNode, builderForNode, readerForNode)
}

func builderForNode() chan types.Value {
	c := make(chan types.Value)
	go func() {
		s := Node{ref: &ref.Ref{}}
		s._Geoposition = (<-c).(Geoposition)
		s._Reference = (<-c).(RefOfValue)
		c <- s
	}()
	return c
}

func readerForNode(v types.Value) chan types.Value {
	c := make(chan types.Value)
	go func() {
		s := v.(Node)
		c <- s._Geoposition
		c <- s._Reference
	}()
	return c
}

func (s Node) Equals(other types.Value) bool {
	return other != nil && __typeRefForNode.Equals(other.TypeRef()) && s.Ref() == other.Ref()
}

func (s Node) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s Node) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, __typeRefForNode.Chunks()...)
	chunks = append(chunks, s._Geoposition.Chunks()...)
	chunks = append(chunks, s._Reference.Chunks()...)
	return
}

func (s Node) Geoposition() Geoposition {
	return s._Geoposition
}

func (s Node) SetGeoposition(val Geoposition) Node {
	s._Geoposition = val
	s.ref = &ref.Ref{}
	return s
}

func (s Node) Reference() RefOfValue {
	return s._Reference
}

func (s Node) SetReference(val RefOfValue) Node {
	s._Reference = val
	s.ref = &ref.Ref{}
	return s
}

// QuadTree

type QuadTree struct {
	_Nodes          ListOfNode
	_Tiles          MapOfStringToQuadTree
	_Depth          uint8
	_NumDescendents uint32
	_Path           string
	_Georectangle   Georectangle

	ref *ref.Ref
}

func NewQuadTree() QuadTree {
	return QuadTree{
		_Nodes:          NewListOfNode(),
		_Tiles:          NewMapOfStringToQuadTree(),
		_Depth:          uint8(0),
		_NumDescendents: uint32(0),
		_Path:           "",
		_Georectangle:   NewGeorectangle(),

		ref: &ref.Ref{},
	}
}

type QuadTreeDef struct {
	Nodes          ListOfNodeDef
	Tiles          MapOfStringToQuadTreeDef
	Depth          uint8
	NumDescendents uint32
	Path           string
	Georectangle   GeorectangleDef
}

func (def QuadTreeDef) New() QuadTree {
	return QuadTree{
		_Nodes:          def.Nodes.New(),
		_Tiles:          def.Tiles.New(),
		_Depth:          def.Depth,
		_NumDescendents: def.NumDescendents,
		_Path:           def.Path,
		_Georectangle:   def.Georectangle.New(),
		ref:             &ref.Ref{},
	}
}

func (s QuadTree) Def() (d QuadTreeDef) {
	d.Nodes = s._Nodes.Def()
	d.Tiles = s._Tiles.Def()
	d.Depth = s._Depth
	d.NumDescendents = s._NumDescendents
	d.Path = s._Path
	d.Georectangle = s._Georectangle.Def()
	return
}

var __typeRefForQuadTree types.TypeRef

func (m QuadTree) TypeRef() types.TypeRef {
	return __typeRefForQuadTree
}

func init() {
	__typeRefForQuadTree = types.MakeTypeRef(__commonPackageInFile_quad_tree_CachedRef, 1)
	types.RegisterStruct(__typeRefForQuadTree, builderForQuadTree, readerForQuadTree)
}

func builderForQuadTree() chan types.Value {
	c := make(chan types.Value)
	go func() {
		s := QuadTree{ref: &ref.Ref{}}
		s._Nodes = (<-c).(ListOfNode)
		s._Tiles = (<-c).(MapOfStringToQuadTree)
		s._Depth = uint8((<-c).(types.UInt8))
		s._NumDescendents = uint32((<-c).(types.UInt32))
		s._Path = (<-c).(types.String).String()
		s._Georectangle = (<-c).(Georectangle)
		c <- s
	}()
	return c
}

func readerForQuadTree(v types.Value) chan types.Value {
	c := make(chan types.Value)
	go func() {
		s := v.(QuadTree)
		c <- s._Nodes
		c <- s._Tiles
		c <- types.UInt8(s._Depth)
		c <- types.UInt32(s._NumDescendents)
		c <- types.NewString(s._Path)
		c <- s._Georectangle
	}()
	return c
}

func (s QuadTree) Equals(other types.Value) bool {
	return other != nil && __typeRefForQuadTree.Equals(other.TypeRef()) && s.Ref() == other.Ref()
}

func (s QuadTree) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s QuadTree) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, __typeRefForQuadTree.Chunks()...)
	chunks = append(chunks, s._Nodes.Chunks()...)
	chunks = append(chunks, s._Tiles.Chunks()...)
	chunks = append(chunks, s._Georectangle.Chunks()...)
	return
}

func (s QuadTree) Nodes() ListOfNode {
	return s._Nodes
}

func (s QuadTree) SetNodes(val ListOfNode) QuadTree {
	s._Nodes = val
	s.ref = &ref.Ref{}
	return s
}

func (s QuadTree) Tiles() MapOfStringToQuadTree {
	return s._Tiles
}

func (s QuadTree) SetTiles(val MapOfStringToQuadTree) QuadTree {
	s._Tiles = val
	s.ref = &ref.Ref{}
	return s
}

func (s QuadTree) Depth() uint8 {
	return s._Depth
}

func (s QuadTree) SetDepth(val uint8) QuadTree {
	s._Depth = val
	s.ref = &ref.Ref{}
	return s
}

func (s QuadTree) NumDescendents() uint32 {
	return s._NumDescendents
}

func (s QuadTree) SetNumDescendents(val uint32) QuadTree {
	s._NumDescendents = val
	s.ref = &ref.Ref{}
	return s
}

func (s QuadTree) Path() string {
	return s._Path
}

func (s QuadTree) SetPath(val string) QuadTree {
	s._Path = val
	s.ref = &ref.Ref{}
	return s
}

func (s QuadTree) Georectangle() Georectangle {
	return s._Georectangle
}

func (s QuadTree) SetGeorectangle(val Georectangle) QuadTree {
	s._Georectangle = val
	s.ref = &ref.Ref{}
	return s
}

// SQuadTree

type SQuadTree struct {
	_Nodes          ListOfRefOfValue
	_Tiles          MapOfStringToRefOfSQuadTree
	_Depth          uint8
	_NumDescendents uint32
	_Path           string
	_Georectangle   Georectangle

	ref *ref.Ref
}

func NewSQuadTree() SQuadTree {
	return SQuadTree{
		_Nodes:          NewListOfRefOfValue(),
		_Tiles:          NewMapOfStringToRefOfSQuadTree(),
		_Depth:          uint8(0),
		_NumDescendents: uint32(0),
		_Path:           "",
		_Georectangle:   NewGeorectangle(),

		ref: &ref.Ref{},
	}
}

type SQuadTreeDef struct {
	Nodes          ListOfRefOfValueDef
	Tiles          MapOfStringToRefOfSQuadTreeDef
	Depth          uint8
	NumDescendents uint32
	Path           string
	Georectangle   GeorectangleDef
}

func (def SQuadTreeDef) New() SQuadTree {
	return SQuadTree{
		_Nodes:          def.Nodes.New(),
		_Tiles:          def.Tiles.New(),
		_Depth:          def.Depth,
		_NumDescendents: def.NumDescendents,
		_Path:           def.Path,
		_Georectangle:   def.Georectangle.New(),
		ref:             &ref.Ref{},
	}
}

func (s SQuadTree) Def() (d SQuadTreeDef) {
	d.Nodes = s._Nodes.Def()
	d.Tiles = s._Tiles.Def()
	d.Depth = s._Depth
	d.NumDescendents = s._NumDescendents
	d.Path = s._Path
	d.Georectangle = s._Georectangle.Def()
	return
}

var __typeRefForSQuadTree types.TypeRef

func (m SQuadTree) TypeRef() types.TypeRef {
	return __typeRefForSQuadTree
}

func init() {
	__typeRefForSQuadTree = types.MakeTypeRef(__commonPackageInFile_quad_tree_CachedRef, 2)
	types.RegisterStruct(__typeRefForSQuadTree, builderForSQuadTree, readerForSQuadTree)
}

func builderForSQuadTree() chan types.Value {
	c := make(chan types.Value)
	go func() {
		s := SQuadTree{ref: &ref.Ref{}}
		s._Nodes = (<-c).(ListOfRefOfValue)
		s._Tiles = (<-c).(MapOfStringToRefOfSQuadTree)
		s._Depth = uint8((<-c).(types.UInt8))
		s._NumDescendents = uint32((<-c).(types.UInt32))
		s._Path = (<-c).(types.String).String()
		s._Georectangle = (<-c).(Georectangle)
		c <- s
	}()
	return c
}

func readerForSQuadTree(v types.Value) chan types.Value {
	c := make(chan types.Value)
	go func() {
		s := v.(SQuadTree)
		c <- s._Nodes
		c <- s._Tiles
		c <- types.UInt8(s._Depth)
		c <- types.UInt32(s._NumDescendents)
		c <- types.NewString(s._Path)
		c <- s._Georectangle
	}()
	return c
}

func (s SQuadTree) Equals(other types.Value) bool {
	return other != nil && __typeRefForSQuadTree.Equals(other.TypeRef()) && s.Ref() == other.Ref()
}

func (s SQuadTree) Ref() ref.Ref {
	return types.EnsureRef(s.ref, s)
}

func (s SQuadTree) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, __typeRefForSQuadTree.Chunks()...)
	chunks = append(chunks, s._Nodes.Chunks()...)
	chunks = append(chunks, s._Tiles.Chunks()...)
	chunks = append(chunks, s._Georectangle.Chunks()...)
	return
}

func (s SQuadTree) Nodes() ListOfRefOfValue {
	return s._Nodes
}

func (s SQuadTree) SetNodes(val ListOfRefOfValue) SQuadTree {
	s._Nodes = val
	s.ref = &ref.Ref{}
	return s
}

func (s SQuadTree) Tiles() MapOfStringToRefOfSQuadTree {
	return s._Tiles
}

func (s SQuadTree) SetTiles(val MapOfStringToRefOfSQuadTree) SQuadTree {
	s._Tiles = val
	s.ref = &ref.Ref{}
	return s
}

func (s SQuadTree) Depth() uint8 {
	return s._Depth
}

func (s SQuadTree) SetDepth(val uint8) SQuadTree {
	s._Depth = val
	s.ref = &ref.Ref{}
	return s
}

func (s SQuadTree) NumDescendents() uint32 {
	return s._NumDescendents
}

func (s SQuadTree) SetNumDescendents(val uint32) SQuadTree {
	s._NumDescendents = val
	s.ref = &ref.Ref{}
	return s
}

func (s SQuadTree) Path() string {
	return s._Path
}

func (s SQuadTree) SetPath(val string) SQuadTree {
	s._Path = val
	s.ref = &ref.Ref{}
	return s
}

func (s SQuadTree) Georectangle() Georectangle {
	return s._Georectangle
}

func (s SQuadTree) SetGeorectangle(val Georectangle) SQuadTree {
	s._Georectangle = val
	s.ref = &ref.Ref{}
	return s
}

// RefOfValue

type RefOfValue struct {
	target ref.Ref
	ref    *ref.Ref
}

func NewRefOfValue(target ref.Ref) RefOfValue {
	return RefOfValue{target, &ref.Ref{}}
}

func (r RefOfValue) TargetRef() ref.Ref {
	return r.target
}

func (r RefOfValue) Ref() ref.Ref {
	return types.EnsureRef(r.ref, r)
}

func (r RefOfValue) Equals(other types.Value) bool {
	return other != nil && __typeRefForRefOfValue.Equals(other.TypeRef()) && r.Ref() == other.Ref()
}

func (r RefOfValue) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, r.TypeRef().Chunks()...)
	chunks = append(chunks, r.target)
	return
}

// A Noms Value that describes RefOfValue.
var __typeRefForRefOfValue types.TypeRef

func (m RefOfValue) TypeRef() types.TypeRef {
	return __typeRefForRefOfValue
}

func init() {
	__typeRefForRefOfValue = types.MakeCompoundTypeRef(types.RefKind, types.MakePrimitiveTypeRef(types.ValueKind))
	types.RegisterFromValFunction(__typeRefForRefOfValue, func(v types.Value) types.Value {
		return NewRefOfValue(v.(types.Ref).TargetRef())
	})
}

func (r RefOfValue) TargetValue(cs chunks.ChunkSource) types.Value {
	return types.ReadValue(r.target, cs)
}

func (r RefOfValue) SetTargetValue(val types.Value, cs chunks.ChunkSink) RefOfValue {
	return NewRefOfValue(types.WriteValue(val, cs))
}

// ListOfNode

type ListOfNode struct {
	l   types.List
	ref *ref.Ref
}

func NewListOfNode() ListOfNode {
	return ListOfNode{types.NewList(), &ref.Ref{}}
}

type ListOfNodeDef []NodeDef

func (def ListOfNodeDef) New() ListOfNode {
	l := make([]types.Value, len(def))
	for i, d := range def {
		l[i] = d.New()
	}
	return ListOfNode{types.NewList(l...), &ref.Ref{}}
}

func (l ListOfNode) Def() ListOfNodeDef {
	d := make([]NodeDef, l.Len())
	for i := uint64(0); i < l.Len(); i++ {
		d[i] = l.l.Get(i).(Node).Def()
	}
	return d
}

func (l ListOfNode) Equals(other types.Value) bool {
	return other != nil && __typeRefForListOfNode.Equals(other.TypeRef()) && l.Ref() == other.Ref()
}

func (l ListOfNode) Ref() ref.Ref {
	return types.EnsureRef(l.ref, l)
}

func (l ListOfNode) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, l.TypeRef().Chunks()...)
	chunks = append(chunks, l.l.Chunks()...)
	return
}

// A Noms Value that describes ListOfNode.
var __typeRefForListOfNode types.TypeRef

func (m ListOfNode) TypeRef() types.TypeRef {
	return __typeRefForListOfNode
}

func init() {
	__typeRefForListOfNode = types.MakeCompoundTypeRef(types.ListKind, types.MakeTypeRef(__commonPackageInFile_quad_tree_CachedRef, 0))
	types.RegisterValue(__typeRefForListOfNode, builderForListOfNode, readerForListOfNode)
}

func builderForListOfNode(v types.Value) types.Value {
	return ListOfNode{v.(types.List), &ref.Ref{}}
}

func readerForListOfNode(v types.Value) types.Value {
	return v.(ListOfNode).l
}

func (l ListOfNode) Len() uint64 {
	return l.l.Len()
}

func (l ListOfNode) Empty() bool {
	return l.Len() == uint64(0)
}

func (l ListOfNode) Get(i uint64) Node {
	return l.l.Get(i).(Node)
}

func (l ListOfNode) Slice(idx uint64, end uint64) ListOfNode {
	return ListOfNode{l.l.Slice(idx, end), &ref.Ref{}}
}

func (l ListOfNode) Set(i uint64, val Node) ListOfNode {
	return ListOfNode{l.l.Set(i, val), &ref.Ref{}}
}

func (l ListOfNode) Append(v ...Node) ListOfNode {
	return ListOfNode{l.l.Append(l.fromElemSlice(v)...), &ref.Ref{}}
}

func (l ListOfNode) Insert(idx uint64, v ...Node) ListOfNode {
	return ListOfNode{l.l.Insert(idx, l.fromElemSlice(v)...), &ref.Ref{}}
}

func (l ListOfNode) Remove(idx uint64, end uint64) ListOfNode {
	return ListOfNode{l.l.Remove(idx, end), &ref.Ref{}}
}

func (l ListOfNode) RemoveAt(idx uint64) ListOfNode {
	return ListOfNode{(l.l.RemoveAt(idx)), &ref.Ref{}}
}

func (l ListOfNode) fromElemSlice(p []Node) []types.Value {
	r := make([]types.Value, len(p))
	for i, v := range p {
		r[i] = v
	}
	return r
}

type ListOfNodeIterCallback func(v Node, i uint64) (stop bool)

func (l ListOfNode) Iter(cb ListOfNodeIterCallback) {
	l.l.Iter(func(v types.Value, i uint64) bool {
		return cb(v.(Node), i)
	})
}

type ListOfNodeIterAllCallback func(v Node, i uint64)

func (l ListOfNode) IterAll(cb ListOfNodeIterAllCallback) {
	l.l.IterAll(func(v types.Value, i uint64) {
		cb(v.(Node), i)
	})
}

type ListOfNodeFilterCallback func(v Node, i uint64) (keep bool)

func (l ListOfNode) Filter(cb ListOfNodeFilterCallback) ListOfNode {
	nl := NewListOfNode()
	l.IterAll(func(v Node, i uint64) {
		if cb(v, i) {
			nl = nl.Append(v)
		}
	})
	return nl
}

// MapOfStringToQuadTree

type MapOfStringToQuadTree struct {
	m   types.Map
	ref *ref.Ref
}

func NewMapOfStringToQuadTree() MapOfStringToQuadTree {
	return MapOfStringToQuadTree{types.NewMap(), &ref.Ref{}}
}

type MapOfStringToQuadTreeDef map[string]QuadTreeDef

func (def MapOfStringToQuadTreeDef) New() MapOfStringToQuadTree {
	kv := make([]types.Value, 0, len(def)*2)
	for k, v := range def {
		kv = append(kv, types.NewString(k), v.New())
	}
	return MapOfStringToQuadTree{types.NewMap(kv...), &ref.Ref{}}
}

func (m MapOfStringToQuadTree) Def() MapOfStringToQuadTreeDef {
	def := make(map[string]QuadTreeDef)
	m.m.Iter(func(k, v types.Value) bool {
		def[k.(types.String).String()] = v.(QuadTree).Def()
		return false
	})
	return def
}

func (m MapOfStringToQuadTree) Equals(other types.Value) bool {
	return other != nil && __typeRefForMapOfStringToQuadTree.Equals(other.TypeRef()) && m.Ref() == other.Ref()
}

func (m MapOfStringToQuadTree) Ref() ref.Ref {
	return types.EnsureRef(m.ref, m)
}

func (m MapOfStringToQuadTree) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, m.TypeRef().Chunks()...)
	chunks = append(chunks, m.m.Chunks()...)
	return
}

// A Noms Value that describes MapOfStringToQuadTree.
var __typeRefForMapOfStringToQuadTree types.TypeRef

func (m MapOfStringToQuadTree) TypeRef() types.TypeRef {
	return __typeRefForMapOfStringToQuadTree
}

func init() {
	__typeRefForMapOfStringToQuadTree = types.MakeCompoundTypeRef(types.MapKind, types.MakePrimitiveTypeRef(types.StringKind), types.MakeTypeRef(__commonPackageInFile_quad_tree_CachedRef, 1))
	types.RegisterValue(__typeRefForMapOfStringToQuadTree, builderForMapOfStringToQuadTree, readerForMapOfStringToQuadTree)
}

func builderForMapOfStringToQuadTree(v types.Value) types.Value {
	return MapOfStringToQuadTree{v.(types.Map), &ref.Ref{}}
}

func readerForMapOfStringToQuadTree(v types.Value) types.Value {
	return v.(MapOfStringToQuadTree).m
}

func (m MapOfStringToQuadTree) Empty() bool {
	return m.m.Empty()
}

func (m MapOfStringToQuadTree) Len() uint64 {
	return m.m.Len()
}

func (m MapOfStringToQuadTree) Has(p string) bool {
	return m.m.Has(types.NewString(p))
}

func (m MapOfStringToQuadTree) Get(p string) QuadTree {
	return m.m.Get(types.NewString(p)).(QuadTree)
}

func (m MapOfStringToQuadTree) MaybeGet(p string) (QuadTree, bool) {
	v, ok := m.m.MaybeGet(types.NewString(p))
	if !ok {
		return NewQuadTree(), false
	}
	return v.(QuadTree), ok
}

func (m MapOfStringToQuadTree) Set(k string, v QuadTree) MapOfStringToQuadTree {
	return MapOfStringToQuadTree{m.m.Set(types.NewString(k), v), &ref.Ref{}}
}

// TODO: Implement SetM?

func (m MapOfStringToQuadTree) Remove(p string) MapOfStringToQuadTree {
	return MapOfStringToQuadTree{m.m.Remove(types.NewString(p)), &ref.Ref{}}
}

type MapOfStringToQuadTreeIterCallback func(k string, v QuadTree) (stop bool)

func (m MapOfStringToQuadTree) Iter(cb MapOfStringToQuadTreeIterCallback) {
	m.m.Iter(func(k, v types.Value) bool {
		return cb(k.(types.String).String(), v.(QuadTree))
	})
}

type MapOfStringToQuadTreeIterAllCallback func(k string, v QuadTree)

func (m MapOfStringToQuadTree) IterAll(cb MapOfStringToQuadTreeIterAllCallback) {
	m.m.IterAll(func(k, v types.Value) {
		cb(k.(types.String).String(), v.(QuadTree))
	})
}

type MapOfStringToQuadTreeFilterCallback func(k string, v QuadTree) (keep bool)

func (m MapOfStringToQuadTree) Filter(cb MapOfStringToQuadTreeFilterCallback) MapOfStringToQuadTree {
	nm := NewMapOfStringToQuadTree()
	m.IterAll(func(k string, v QuadTree) {
		if cb(k, v) {
			nm = nm.Set(k, v)
		}
	})
	return nm
}

// ListOfRefOfValue

type ListOfRefOfValue struct {
	l   types.List
	ref *ref.Ref
}

func NewListOfRefOfValue() ListOfRefOfValue {
	return ListOfRefOfValue{types.NewList(), &ref.Ref{}}
}

type ListOfRefOfValueDef []ref.Ref

func (def ListOfRefOfValueDef) New() ListOfRefOfValue {
	l := make([]types.Value, len(def))
	for i, d := range def {
		l[i] = NewRefOfValue(d)
	}
	return ListOfRefOfValue{types.NewList(l...), &ref.Ref{}}
}

func (l ListOfRefOfValue) Def() ListOfRefOfValueDef {
	d := make([]ref.Ref, l.Len())
	for i := uint64(0); i < l.Len(); i++ {
		d[i] = l.l.Get(i).(RefOfValue).TargetRef()
	}
	return d
}

func (l ListOfRefOfValue) Equals(other types.Value) bool {
	return other != nil && __typeRefForListOfRefOfValue.Equals(other.TypeRef()) && l.Ref() == other.Ref()
}

func (l ListOfRefOfValue) Ref() ref.Ref {
	return types.EnsureRef(l.ref, l)
}

func (l ListOfRefOfValue) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, l.TypeRef().Chunks()...)
	chunks = append(chunks, l.l.Chunks()...)
	return
}

// A Noms Value that describes ListOfRefOfValue.
var __typeRefForListOfRefOfValue types.TypeRef

func (m ListOfRefOfValue) TypeRef() types.TypeRef {
	return __typeRefForListOfRefOfValue
}

func init() {
	__typeRefForListOfRefOfValue = types.MakeCompoundTypeRef(types.ListKind, types.MakeCompoundTypeRef(types.RefKind, types.MakePrimitiveTypeRef(types.ValueKind)))
	types.RegisterValue(__typeRefForListOfRefOfValue, builderForListOfRefOfValue, readerForListOfRefOfValue)
}

func builderForListOfRefOfValue(v types.Value) types.Value {
	return ListOfRefOfValue{v.(types.List), &ref.Ref{}}
}

func readerForListOfRefOfValue(v types.Value) types.Value {
	return v.(ListOfRefOfValue).l
}

func (l ListOfRefOfValue) Len() uint64 {
	return l.l.Len()
}

func (l ListOfRefOfValue) Empty() bool {
	return l.Len() == uint64(0)
}

func (l ListOfRefOfValue) Get(i uint64) RefOfValue {
	return l.l.Get(i).(RefOfValue)
}

func (l ListOfRefOfValue) Slice(idx uint64, end uint64) ListOfRefOfValue {
	return ListOfRefOfValue{l.l.Slice(idx, end), &ref.Ref{}}
}

func (l ListOfRefOfValue) Set(i uint64, val RefOfValue) ListOfRefOfValue {
	return ListOfRefOfValue{l.l.Set(i, val), &ref.Ref{}}
}

func (l ListOfRefOfValue) Append(v ...RefOfValue) ListOfRefOfValue {
	return ListOfRefOfValue{l.l.Append(l.fromElemSlice(v)...), &ref.Ref{}}
}

func (l ListOfRefOfValue) Insert(idx uint64, v ...RefOfValue) ListOfRefOfValue {
	return ListOfRefOfValue{l.l.Insert(idx, l.fromElemSlice(v)...), &ref.Ref{}}
}

func (l ListOfRefOfValue) Remove(idx uint64, end uint64) ListOfRefOfValue {
	return ListOfRefOfValue{l.l.Remove(idx, end), &ref.Ref{}}
}

func (l ListOfRefOfValue) RemoveAt(idx uint64) ListOfRefOfValue {
	return ListOfRefOfValue{(l.l.RemoveAt(idx)), &ref.Ref{}}
}

func (l ListOfRefOfValue) fromElemSlice(p []RefOfValue) []types.Value {
	r := make([]types.Value, len(p))
	for i, v := range p {
		r[i] = v
	}
	return r
}

type ListOfRefOfValueIterCallback func(v RefOfValue, i uint64) (stop bool)

func (l ListOfRefOfValue) Iter(cb ListOfRefOfValueIterCallback) {
	l.l.Iter(func(v types.Value, i uint64) bool {
		return cb(v.(RefOfValue), i)
	})
}

type ListOfRefOfValueIterAllCallback func(v RefOfValue, i uint64)

func (l ListOfRefOfValue) IterAll(cb ListOfRefOfValueIterAllCallback) {
	l.l.IterAll(func(v types.Value, i uint64) {
		cb(v.(RefOfValue), i)
	})
}

type ListOfRefOfValueFilterCallback func(v RefOfValue, i uint64) (keep bool)

func (l ListOfRefOfValue) Filter(cb ListOfRefOfValueFilterCallback) ListOfRefOfValue {
	nl := NewListOfRefOfValue()
	l.IterAll(func(v RefOfValue, i uint64) {
		if cb(v, i) {
			nl = nl.Append(v)
		}
	})
	return nl
}

// MapOfStringToRefOfSQuadTree

type MapOfStringToRefOfSQuadTree struct {
	m   types.Map
	ref *ref.Ref
}

func NewMapOfStringToRefOfSQuadTree() MapOfStringToRefOfSQuadTree {
	return MapOfStringToRefOfSQuadTree{types.NewMap(), &ref.Ref{}}
}

type MapOfStringToRefOfSQuadTreeDef map[string]ref.Ref

func (def MapOfStringToRefOfSQuadTreeDef) New() MapOfStringToRefOfSQuadTree {
	kv := make([]types.Value, 0, len(def)*2)
	for k, v := range def {
		kv = append(kv, types.NewString(k), NewRefOfSQuadTree(v))
	}
	return MapOfStringToRefOfSQuadTree{types.NewMap(kv...), &ref.Ref{}}
}

func (m MapOfStringToRefOfSQuadTree) Def() MapOfStringToRefOfSQuadTreeDef {
	def := make(map[string]ref.Ref)
	m.m.Iter(func(k, v types.Value) bool {
		def[k.(types.String).String()] = v.(RefOfSQuadTree).TargetRef()
		return false
	})
	return def
}

func (m MapOfStringToRefOfSQuadTree) Equals(other types.Value) bool {
	return other != nil && __typeRefForMapOfStringToRefOfSQuadTree.Equals(other.TypeRef()) && m.Ref() == other.Ref()
}

func (m MapOfStringToRefOfSQuadTree) Ref() ref.Ref {
	return types.EnsureRef(m.ref, m)
}

func (m MapOfStringToRefOfSQuadTree) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, m.TypeRef().Chunks()...)
	chunks = append(chunks, m.m.Chunks()...)
	return
}

// A Noms Value that describes MapOfStringToRefOfSQuadTree.
var __typeRefForMapOfStringToRefOfSQuadTree types.TypeRef

func (m MapOfStringToRefOfSQuadTree) TypeRef() types.TypeRef {
	return __typeRefForMapOfStringToRefOfSQuadTree
}

func init() {
	__typeRefForMapOfStringToRefOfSQuadTree = types.MakeCompoundTypeRef(types.MapKind, types.MakePrimitiveTypeRef(types.StringKind), types.MakeCompoundTypeRef(types.RefKind, types.MakeTypeRef(__commonPackageInFile_quad_tree_CachedRef, 2)))
	types.RegisterValue(__typeRefForMapOfStringToRefOfSQuadTree, builderForMapOfStringToRefOfSQuadTree, readerForMapOfStringToRefOfSQuadTree)
}

func builderForMapOfStringToRefOfSQuadTree(v types.Value) types.Value {
	return MapOfStringToRefOfSQuadTree{v.(types.Map), &ref.Ref{}}
}

func readerForMapOfStringToRefOfSQuadTree(v types.Value) types.Value {
	return v.(MapOfStringToRefOfSQuadTree).m
}

func (m MapOfStringToRefOfSQuadTree) Empty() bool {
	return m.m.Empty()
}

func (m MapOfStringToRefOfSQuadTree) Len() uint64 {
	return m.m.Len()
}

func (m MapOfStringToRefOfSQuadTree) Has(p string) bool {
	return m.m.Has(types.NewString(p))
}

func (m MapOfStringToRefOfSQuadTree) Get(p string) RefOfSQuadTree {
	return m.m.Get(types.NewString(p)).(RefOfSQuadTree)
}

func (m MapOfStringToRefOfSQuadTree) MaybeGet(p string) (RefOfSQuadTree, bool) {
	v, ok := m.m.MaybeGet(types.NewString(p))
	if !ok {
		return NewRefOfSQuadTree(ref.Ref{}), false
	}
	return v.(RefOfSQuadTree), ok
}

func (m MapOfStringToRefOfSQuadTree) Set(k string, v RefOfSQuadTree) MapOfStringToRefOfSQuadTree {
	return MapOfStringToRefOfSQuadTree{m.m.Set(types.NewString(k), v), &ref.Ref{}}
}

// TODO: Implement SetM?

func (m MapOfStringToRefOfSQuadTree) Remove(p string) MapOfStringToRefOfSQuadTree {
	return MapOfStringToRefOfSQuadTree{m.m.Remove(types.NewString(p)), &ref.Ref{}}
}

type MapOfStringToRefOfSQuadTreeIterCallback func(k string, v RefOfSQuadTree) (stop bool)

func (m MapOfStringToRefOfSQuadTree) Iter(cb MapOfStringToRefOfSQuadTreeIterCallback) {
	m.m.Iter(func(k, v types.Value) bool {
		return cb(k.(types.String).String(), v.(RefOfSQuadTree))
	})
}

type MapOfStringToRefOfSQuadTreeIterAllCallback func(k string, v RefOfSQuadTree)

func (m MapOfStringToRefOfSQuadTree) IterAll(cb MapOfStringToRefOfSQuadTreeIterAllCallback) {
	m.m.IterAll(func(k, v types.Value) {
		cb(k.(types.String).String(), v.(RefOfSQuadTree))
	})
}

type MapOfStringToRefOfSQuadTreeFilterCallback func(k string, v RefOfSQuadTree) (keep bool)

func (m MapOfStringToRefOfSQuadTree) Filter(cb MapOfStringToRefOfSQuadTreeFilterCallback) MapOfStringToRefOfSQuadTree {
	nm := NewMapOfStringToRefOfSQuadTree()
	m.IterAll(func(k string, v RefOfSQuadTree) {
		if cb(k, v) {
			nm = nm.Set(k, v)
		}
	})
	return nm
}

// RefOfSQuadTree

type RefOfSQuadTree struct {
	target ref.Ref
	ref    *ref.Ref
}

func NewRefOfSQuadTree(target ref.Ref) RefOfSQuadTree {
	return RefOfSQuadTree{target, &ref.Ref{}}
}

func (r RefOfSQuadTree) TargetRef() ref.Ref {
	return r.target
}

func (r RefOfSQuadTree) Ref() ref.Ref {
	return types.EnsureRef(r.ref, r)
}

func (r RefOfSQuadTree) Equals(other types.Value) bool {
	return other != nil && __typeRefForRefOfSQuadTree.Equals(other.TypeRef()) && r.Ref() == other.Ref()
}

func (r RefOfSQuadTree) Chunks() (chunks []ref.Ref) {
	chunks = append(chunks, r.TypeRef().Chunks()...)
	chunks = append(chunks, r.target)
	return
}

// A Noms Value that describes RefOfSQuadTree.
var __typeRefForRefOfSQuadTree types.TypeRef

func (m RefOfSQuadTree) TypeRef() types.TypeRef {
	return __typeRefForRefOfSQuadTree
}

func init() {
	__typeRefForRefOfSQuadTree = types.MakeCompoundTypeRef(types.RefKind, types.MakeTypeRef(__commonPackageInFile_quad_tree_CachedRef, 2))
	types.RegisterFromValFunction(__typeRefForRefOfSQuadTree, func(v types.Value) types.Value {
		return NewRefOfSQuadTree(v.(types.Ref).TargetRef())
	})
}

func (r RefOfSQuadTree) TargetValue(cs chunks.ChunkSource) SQuadTree {
	return types.ReadValue(r.target, cs).(SQuadTree)
}

func (r RefOfSQuadTree) SetTargetValue(val SQuadTree, cs chunks.ChunkSink) RefOfSQuadTree {
	return NewRefOfSQuadTree(types.WriteValue(val, cs))
}
